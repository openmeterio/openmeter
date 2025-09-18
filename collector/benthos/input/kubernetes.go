package input

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // import kubernetes auth plugins
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/openmeterio/openmeter/collector/benthos/internal/logging"
	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

// TODO: add batching config and policy

func kubernetesResourcesInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("List resources in Kubernetes.").
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

	manager manager.Manager
	client  client.Client

	// Used for graceful shutdown of the manager.
	cancel context.CancelFunc
	done   chan struct{}
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
		namespaces:         namespaces,
		labelSelector:      selector,
		resourceType:       resourceType,
		includePendingPods: includePendingPods,
		manager:            mgr,
		client:             client,
		logger:             logger,
		resources:          res,
	}, nil
}

func (in *kubernetesResourcesInput) Connect(ctx context.Context) error {
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

func (in *kubernetesResourcesInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	batch := make([]*service.Message, 0)

	if !leaderelection.IsLeader(in.resources) {
		return batch, func(context.Context, error) error { return nil }, nil
	}

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
					// Skip pods in other phases (Succeeded, Failed, Unknown)
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

	in.logger.Debugf("batch size of %s: %d", in.resourceType, len(batch))

	return batch, func(context.Context, error) error {
		// A nack (when err is non-nil) is handled automatically when we
		// construct using service.AutoRetryNacks.
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
