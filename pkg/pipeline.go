package pkg

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/rs/zerolog/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetKubeClient ...
func GetKubeClient() (client *kubernetes.Clientset, config *rest.Config, err error) {

	config, err = ctrl.GetConfig()
	if err != nil {
		return client, config, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, config, err
	}

	return clientset, config, err
}

// GetPipeline ...
func GetPipeline() (v1alpha1.Pipeline, error) {

	instance := v1alpha1.Pipeline{}
	ns := os.Getenv("CHURRO_NAMESPACE")
	pipelineName := os.Getenv("CHURRO_PIPELINE")
	if ns == "" {
		return instance, errors.New("CHURRO_PIPELINE not set")
	}
	if pipelineName == "" {
		return instance, errors.New("CHURRO_PIPELINE not set")
	}
	log.Info().Msg("CHURRO_NAMESPACE " + ns)
	log.Info().Msg("CHURRO_PIPELINE " + pipelineName)

	_, cfg, err := GetKubeClient()
	if err != nil {
		return instance, err
	}

	err = v1alpha1.AddToScheme(clientgoscheme.Scheme)
	if err != nil {
		return instance, err
	}

	var k8sClient client.Client
	k8sClient, err = client.New(cfg, client.Options{Scheme: clientgoscheme.Scheme})
	if err != nil {
		return instance, err
	}

	namespacedName := types.NamespacedName{
		Namespace: ns,
		Name:      pipelineName,
	}

	err = k8sClient.Get(context.TODO(), namespacedName, &instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// SchemeGroupVersion ...
var SchemeGroupVersion = schema.GroupVersion{Group: "churro.project.io", Version: "v1alpha1"}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1alpha1.Pipeline{},
		&v1alpha1.PipelineList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// NewClient ...
func NewClient(cfg *rest.Config, namespace string) (*PipelineClient, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	config := *cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PipelineClient{restClient: client, ns: namespace}, nil
}

// PipelineClient ...
type PipelineClient struct {
	restClient rest.Interface
	ns         string
}

// Get ....
func (c *PipelineClient) Get(name string) (*v1alpha1.Pipeline, error) {
	result := &v1alpha1.Pipeline{}
	err := c.restClient.Get().
		Namespace(c.ns).Resource("pipelines").
		Name(name).Do(context.TODO()).Into(result)
	return result, err
}

// List ....
func (c *PipelineClient) List() (*v1alpha1.PipelineList, error) {
	result := &v1alpha1.PipelineList{}
	err := c.restClient.Get().
		Resource("pipelines").
		Do(context.TODO()).Into(result)
	return result, err
}

// Create ....
func (c *PipelineClient) Create(obj *v1alpha1.Pipeline) (*v1alpha1.Pipeline, error) {
	result := &v1alpha1.Pipeline{}
	err := c.restClient.Post().
		Namespace(c.ns).Resource("pipelines").
		Body(obj).Do(context.TODO()).Into(result)
	return result, err
}

// Update ...
func (c *PipelineClient) Update(obj *v1alpha1.Pipeline) (*v1alpha1.Pipeline, error) {
	log.Info().Msg(fmt.Sprintf("updating pipeline with %v\n", obj))
	result := &v1alpha1.Pipeline{}
	err := c.restClient.Put().
		Namespace(c.ns).Resource("pipelines").Name(obj.Name).
		Body(obj).Do(context.TODO()).Into(result)
	if err != nil {
		log.Error().Stack().Err(err)
	}
	return result, err
}

// Delete ...
func (c *PipelineClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).Resource("pipelines").
		Name(name).Body(options).Do(context.TODO()).
		Error()
}
