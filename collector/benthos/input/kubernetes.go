package input

import (
	"context"
	"strings"

	"github.com/benthosdev/benthos/v4/public/service"
	"github.com/samber/lo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // import kubernetes auth plugins
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// TODO: add batching config and policy

func kubernetesResourcesInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("List objects in Kubernetes.").
		Fields(
			service.NewObjectField(
				"resource",
				service.NewStringField("group").
					Description("Kubernetes API group.").
					Optional(),
				service.NewStringField("version").
					Description("Kubernetes API group version.").
					Example("v1"),
				service.NewStringField("name").
					Description("Kubernetes API resource name.").
					Example("pods"),
			).
				Description("Kubernetes resource details.").
				Advanced(),
			service.NewStringListField("namespaces").
				Description("List of namespaces to list objects from."),
			service.NewStringField("label_selector").
				Description("Label selector applied to each list operation.").
				Optional(),
		)
}

func init() {
	err := service.RegisterBatchInput("kubernetes_resources", kubernetesResourcesInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		return newKubernetesResourcesInput(conf)
	})
	if err != nil {
		panic(err)
	}
}

type kubernetesResourcesInput struct {
	gvr           schema.GroupVersionResource
	namespaces    []string
	labelSelector string

	client dynamic.Interface
}

func newKubernetesResourcesInput(conf *service.ParsedConfig) (*kubernetesResourcesInput, error) {
	var gvr schema.GroupVersionResource

	{
		conf := conf.Namespace("resource")

		var err error

		if conf.Contains("group") {
			if gvr.Group, err = conf.FieldString("group"); err != nil {
				return nil, err
			}
		}

		if gvr.Version, err = conf.FieldString("version"); err != nil {
			return nil, err
		}

		if gvr.Resource, err = conf.FieldString("name"); err != nil {
			return nil, err
		}
	}

	namespaces, err := conf.FieldStringList("namespaces")
	if err != nil {
		return nil, err
	}

	namespaces = lo.Uniq(lo.Map(namespaces, func(s string, _ int) string { return strings.ToLower(s) }))

	var labelSelector string

	if conf.Contains("label_selector") {
		if labelSelector, err = conf.FieldString("label_selector"); err != nil {
			return nil, err
		}
	}

	kubeconfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &kubernetesResourcesInput{
		gvr:           gvr,
		namespaces:    namespaces,
		labelSelector: labelSelector,
		client:        client,
	}, nil
}

func (in *kubernetesResourcesInput) Connect(_ context.Context) error {
	return nil
}

func (in *kubernetesResourcesInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	var resources []unstructured.Unstructured

	for _, namespace := range in.namespaces {
		list, err := in.client.Resource(in.gvr).Namespace(namespace).List(ctx, v1.ListOptions{
			LabelSelector: in.labelSelector,
		})
		if err != nil {
			return nil, nil, err
		}

		resources = append(resources, list.Items...)
	}

	batch := make([]*service.Message, 0, len(resources))

	for _, resource := range resources {
		encoded, err := resource.MarshalJSON()
		if err != nil { // TODO: better error handling
			return nil, nil, err
		}

		batch = append(batch, service.NewMessage(encoded))
	}

	return batch, func(context.Context, error) error { return nil }, nil
}

func (in *kubernetesResourcesInput) Close(_ context.Context) error {
	return nil
}
