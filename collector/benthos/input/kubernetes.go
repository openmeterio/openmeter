package input

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // import kubernetes auth plugins
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/openmeterio/openmeter/collector/benthos/internal/logging"
	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

// TODO: add batching config and policy

// maxDeletedPods bounds the deleted pods kept between two reads. When the
// pipeline is blocked no read happens, and the oldest entries are dropped.
const maxDeletedPods = 10000

func kubernetesResourcesInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("List resources in Kubernetes.").
		Description(`
Each message carries the following metadata:

- `+"`kubernetes_read_time`"+`: the time the resources were read (RFC 3339). A batch is read as soon as the previous one was acknowledged or rejected, which can be some time before it is handed out: the `+"`schedule`"+` input holds it until the next tick.
- `+"`kubernetes_last_ack_read_time`"+`: the `+"`kubernetes_read_time`"+` of the latest batch that was acknowledged without an error. It is not set until a batch was acknowledged. A mapping can use it to cover the whole time since the last delivered read, so a rejected batch or a skipped read leaves no gap.
- `+"`kubernetes_deleted_time`"+`: only on a deleted pod (see `+"`include_deleted_pods`"+`), the time the deletion was observed (RFC 3339).
`).
		Fields(
			service.NewStringListField("namespaces").
				Description("List of namespaces to list resources from."),
			service.NewStringEnumField("resource_type", "pod", "node", "persistentvolume", "persistentvolumeclaim").
				Description("Type of resource to list.").
				Default("pod"),
			service.NewStringField("label_selector").
				Description("Label selector applied to each list operation.").
				Optional(),
			service.NewBoolField("include_pending_pods").
				Description("Include pods in pending state (not all containers are running). Only applies when resource_type is 'pod'.").
				Default(false),
			service.NewBoolField("include_terminated_pods").
				Description("Include pods in a terminal phase (Succeeded or Failed) for as long as they exist. Only applies when resource_type is 'pod'.").
				Default(false),
			service.NewBoolField("include_deleted_pods").
				Description("Emit each pod once more after it is deleted, in its last known state, with the metadata field `kubernetes_deleted_time`. Lets a mapping account for the time between the previous read and the deletion. Only applies when resource_type is 'pod'.").
				Default(false),
		)
}

func init() {
	err := service.RegisterBatchInput("kubernetes_resources", kubernetesResourcesInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		return newKubernetesResourcesInput(conf, mgr)
	})
	if err != nil {
		panic(err)
	}
}

type kubernetesResourcesInput struct {
	namespaces         []string
	resourceType       string
	labelSelector      labels.Selector
	includePendingPods bool
	logger             *service.Logger
	resources          *service.Resources

	includeTerminatedPods bool
	includeDeletedPods    bool

	manager manager.Manager
	client  client.Client

	// Holds one token while a batch is neither acknowledged nor rejected.
	inFlight chan struct{}

	mu sync.Mutex
	// Pods the cache saw deleted since the last read.
	deletedPods []deletedPod
	// Read time of the latest batch that was acknowledged without an error.
	lastAckReadTime time.Time

	// Used for graceful shutdown of the manager.
	cancel context.CancelFunc
	done   chan struct{}
}

// deletedPod is the last known state of a deleted pod.
type deletedPod struct {
	pod       *corev1.Pod
	deletedAt time.Time
}

func newKubernetesResourcesInput(conf *service.ParsedConfig, res *service.Resources) (*kubernetesResourcesInput, error) {
	logger := res.Logger().With("component", "kubernetes_resources")

	ctrlLogger := logging.NewLogrLogger(logger)
	ctrllog.SetLogger(ctrlLogger)

	namespaces, err := conf.FieldStringList("namespaces")
	if err != nil {
		return nil, err
	}

	resourceType, err := conf.FieldString("resource_type")
	if err != nil {
		return nil, err
	}

	includePendingPods, err := conf.FieldBool("include_pending_pods")
	if err != nil {
		return nil, err
	}

	includeTerminatedPods, err := conf.FieldBool("include_terminated_pods")
	if err != nil {
		return nil, err
	}

	includeDeletedPods, err := conf.FieldBool("include_deleted_pods")
	if err != nil {
		return nil, err
	}

	// Normalize the namespaces to lowercase and deduplicate.
	namespaces = lo.Uniq(lo.Map(namespaces, func(s string, _ int) string { return strings.ToLower(s) }))

	// If no namespaces are provided, use all namespaces.
	if len(namespaces) == 0 {
		namespaces = []string{corev1.NamespaceAll}
	}

	// Convert a non-empty label selector into a labels.Selector.
	var selector labels.Selector
	if conf.Contains("label_selector") {
		labelSelector, err := conf.FieldString("label_selector")
		if err != nil {
			return nil, err
		}

		selector, err = labels.Parse(labelSelector)
		if err != nil {
			return nil, err
		}
	}

	// Get the kubeconfig.
	kubeconfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// Build a scheme and register core/v1 (pods) into it.
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	// Create a new manager. Its client will automatically use a cache.
	mgr, err := manager.New(kubeconfig, manager.Options{
		Logger: ctrlLogger,
		Scheme: scheme,
		// Disable servers.
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		PprofBindAddress:       "0",
	})
	if err != nil {
		return nil, err
	}

	// Get Kubernetes client.
	client := mgr.GetClient()

	return &kubernetesResourcesInput{
		namespaces:            namespaces,
		labelSelector:         selector,
		resourceType:          resourceType,
		includePendingPods:    includePendingPods,
		includeTerminatedPods: includeTerminatedPods,
		includeDeletedPods:    includeDeletedPods,
		inFlight:              make(chan struct{}, 1),
		manager:               mgr,
		client:                client,
		logger:                logger,
		resources:             res,
	}, nil
}

func (in *kubernetesResourcesInput) Connect(ctx context.Context) error {
	// Watch for deleted pods. The handler is registered before the manager
	// starts, so no deletion is missed once the cache is in sync.
	if in.resourceType == "pod" && in.includeDeletedPods {
		informer, err := in.manager.GetCache().GetInformer(ctx, &corev1.Pod{})
		if err != nil {
			return err
		}

		if _, err := informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
			DeleteFunc: in.onPodDeleted,
		}); err != nil {
			return err
		}
	}

	// Create a cancellable context for the manager.
	mgrCtx, cancel := context.WithCancel(ctx)
	in.cancel = cancel
	in.done = make(chan struct{})

	// Start the manager in a separate goroutine.
	go func() {
		defer close(in.done)

		// Start blocks, so we run it in the background.
		if err := in.manager.Start(mgrCtx); err != nil {
			in.logger.Errorf("failed to start manager: %v", err)
			in.Close(ctx)
		}
	}()

	// Wait for the cache to sync. This ensures that subsequent List() calls
	// use up-to-date cached data.
	if synced := in.manager.GetCache().WaitForCacheSync(ctx); !synced {
		return errors.New("failed to sync cache")
	}

	return nil
}

// onPodDeleted keeps the last known state of a deleted pod until the next read.
func (in *kubernetesResourcesInput) onPodDeleted(obj any) {
	// The cache missed the delete event and only knows a possibly stale state.
	if tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	// Only the leader reads, so other replicas have nothing to hand over.
	if !leaderelection.IsLeader(in.resources) {
		return
	}

	if !lo.Contains(in.namespaces, corev1.NamespaceAll) && !lo.Contains(in.namespaces, pod.Namespace) {
		return
	}

	if in.labelSelector != nil && !in.labelSelector.Matches(labels.Set(pod.Labels)) {
		return
	}

	in.requeueDeletedPods([]deletedPod{{pod: pod, deletedAt: time.Now().UTC()}})
}

// requeueDeletedPods adds deleted pods to hand over with the next read.
func (in *kubernetesResourcesInput) requeueDeletedPods(pods []deletedPod) {
	in.mu.Lock()
	defer in.mu.Unlock()

	in.deletedPods = append(in.deletedPods, pods...)

	if dropped := len(in.deletedPods) - maxDeletedPods; dropped > 0 {
		in.logger.Warnf("dropping %d deleted pods that were not read in time", dropped)
		in.deletedPods = in.deletedPods[dropped:]
	}
}

// takeDeletedPods returns the deleted pods seen since the last read.
func (in *kubernetesResourcesInput) takeDeletedPods() []deletedPod {
	in.mu.Lock()
	defer in.mu.Unlock()

	pods := in.deletedPods
	in.deletedPods = nil

	return pods
}

func (in *kubernetesResourcesInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	batch := make([]*service.Message, 0)

	// Read the next batch only once the previous one was acknowledged or
	// rejected. A batch read earlier would not know what was delivered.
	select {
	case in.inFlight <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	release := func() {
		select {
		case <-in.inFlight:
		default:
		}
	}

	// An empty batch is never acknowledged, and neither is a failed read.
	handedOver := false

	defer func() {
		if !handedOver {
			release()
		}
	}()

	if !leaderelection.IsLeader(in.resources) {
		// Deletions seen while this replica was the leader are not handed over.
		in.takeDeletedPods()

		return batch, func(context.Context, error) error { return nil }, nil
	}

	// Taken before the resources are listed, so they are at least this recent.
	readTime := time.Now().UTC()

	in.mu.Lock()
	lastAckReadTime := in.lastAckReadTime
	in.mu.Unlock()

	// Iterate over each namespace and list pods.
	for _, ns := range in.namespaces {
		opts := []client.ListOption{client.InNamespace(ns)}
		if in.labelSelector != nil {
			opts = append(opts, client.MatchingLabelsSelector{
				Selector: in.labelSelector,
			})
		}

		switch in.resourceType {
		case "pod":
			podList := &corev1.PodList{}
			if err := in.client.List(ctx, podList, opts...); err != nil {
				return nil, nil, err
			}

			for _, pod := range podList.Items {
				// Filter pods based on their status and configuration.
				shouldInclude := false

				// Most recently observed status of the pod. This data may not be up to date. Populated by the system.
				switch pod.Status.Phase {
				case corev1.PodRunning:
					shouldInclude = true
				case corev1.PodPending:
					shouldInclude = in.includePendingPods
				case corev1.PodSucceeded, corev1.PodFailed:
					shouldInclude = in.includeTerminatedPods
				// Phase not set, check container statuses
				case "":
					in.logger.Warnf("pod %s has no phase", pod.Name)
					// If all containers are running, treat as running pod
					if lo.EveryBy(pod.Status.ContainerStatuses, func(cs corev1.ContainerStatus) bool {
						return cs.State.Running != nil
					}) {
						shouldInclude = true
						// If at least one container is running, treat as pending
					} else if lo.SomeBy(pod.Status.ContainerStatuses, func(cs corev1.ContainerStatus) bool {
						return cs.State.Running != nil
					}) {
						shouldInclude = in.includePendingPods
					} else {
						shouldInclude = false
					}
				default:
					in.logger.Warnf("pod %s has unknown phase", pod.Name)
					// Skip pods in other phases (Unknown)
					shouldInclude = false
				}

				if !shouldInclude {
					continue
				}

				encoded, err := json.Marshal(pod)
				if err != nil {
					return nil, nil, err
				}

				in.logger.Debugf("adding pod %s to batch", pod.Name)
				batch = append(batch, service.NewMessage(encoded))
			}
		case "node":
			nodeList := &corev1.NodeList{}
			if err := in.client.List(ctx, nodeList, opts...); err != nil {
				return nil, nil, err
			}

			for _, node := range nodeList.Items {
				encoded, err := json.Marshal(node)
				if err != nil {
					return nil, nil, err
				}

				in.logger.Debugf("adding node %s to batch", node.Name)
				batch = append(batch, service.NewMessage(encoded))
			}
		case "persistentvolume":
			persistentVolumeList := &corev1.PersistentVolumeList{}
			if err := in.client.List(ctx, persistentVolumeList, opts...); err != nil {
				return nil, nil, err
			}

			for _, persistentVolume := range persistentVolumeList.Items {
				encoded, err := json.Marshal(persistentVolume)
				if err != nil {
					return nil, nil, err
				}

				in.logger.Debugf("adding persistent volume %s to batch", persistentVolume.Name)
				batch = append(batch, service.NewMessage(encoded))
			}
		case "persistentvolumeclaim":
			persistentVolumeClaimList := &corev1.PersistentVolumeClaimList{}
			if err := in.client.List(ctx, persistentVolumeClaimList, opts...); err != nil {
				return nil, nil, err
			}

			for _, persistentVolumeClaim := range persistentVolumeClaimList.Items {
				encoded, err := json.Marshal(persistentVolumeClaim)
				if err != nil {
					return nil, nil, err
				}

				in.logger.Debugf("adding persistent volume claim %s to batch", persistentVolumeClaim.Name)
				batch = append(batch, service.NewMessage(encoded))
			}
		}
	}

	// Hand over the pods deleted since the last read, each exactly once.
	deletedPods := in.takeDeletedPods()

	for _, deleted := range deletedPods {
		encoded, err := json.Marshal(deleted.pod)
		if err != nil {
			in.requeueDeletedPods(deletedPods)

			return nil, nil, err
		}

		msg := service.NewMessage(encoded)
		msg.MetaSet("kubernetes_deleted_time", deleted.deletedAt.Format(time.RFC3339))

		in.logger.Debugf("adding deleted pod %s to batch", deleted.pod.Name)
		batch = append(batch, msg)
	}

	for _, msg := range batch {
		msg.MetaSet("kubernetes_read_time", readTime.Format(time.RFC3339))

		if !lastAckReadTime.IsZero() {
			msg.MetaSet("kubernetes_last_ack_read_time", lastAckReadTime.Format(time.RFC3339))
		}
	}

	in.logger.Debugf("batch size of %s: %d", in.resourceType, len(batch))

	handedOver = len(batch) > 0

	return batch, func(_ context.Context, err error) error {
		defer release()

		// The input is not wrapped in service.AutoRetryNacks, so a rejected
		// batch is not sent again. Listed resources are read again next time;
		// deleted pods are gone from the cache, so keep them.
		if err != nil {
			in.requeueDeletedPods(deletedPods)

			return nil
		}

		in.mu.Lock()
		defer in.mu.Unlock()

		in.lastAckReadTime = readTime

		return nil
	}, nil
}

func (in *kubernetesResourcesInput) Close(ctx context.Context) error {
	if in.cancel != nil {
		// Trigger graceful shutdown of the manager
		in.cancel()
	}

	// Wait for the manager's goroutine to exit or for the context to be canceled.
	select {
	case <-in.done:
		in.logger.Info("manager exited")
	case <-ctx.Done():
		in.logger.Info("context canceled")
		return ctx.Err()
	}

	return nil
}
