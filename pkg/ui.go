package pkg

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/churrodata/churro/api/uiv1alpha1"
	"github.com/rs/zerolog/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetChurroui ...
func GetChurroui() (uiv1alpha1.Churroui, error) {

	instance := uiv1alpha1.Churroui{}
	ns := os.Getenv("CHURRO_NAMESPACE")
	crName := os.Getenv("CHURRO_UI_RESOURCE")
	if ns == "" {
		return instance, errors.New("CHURRO_UI_RESOURCE not set")
	}
	if crName == "" {
		return instance, errors.New("CHURRO_UI_RESOURCE not set")
	}
	log.Info().Msg("CHURRO_NAMESPACE " + ns)
	log.Info().Msg("CHURRO_UI_RESOURCE " + crName)

	_, cfg, err := GetKubeClient()
	if err != nil {
		return instance, err
	}

	err = uiv1alpha1.AddToScheme(clientgoscheme.Scheme)
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
		Name:      crName,
	}

	err = k8sClient.Get(context.TODO(), namespacedName, &instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// SchemeGroupVersion ...
var UISchemeGroupVersion = schema.GroupVersion{Group: "churro.project.io", Version: "uiv1alpha1"}

func addKnownTypesUI(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(UISchemeGroupVersion,
		&uiv1alpha1.Churroui{},
		&uiv1alpha1.ChurrouiList{},
	)
	metav1.AddToGroupVersion(scheme, UISchemeGroupVersion)
	return nil
}

// NewUIClient ...
func NewUIClient(cfg *rest.Config, namespace string) (*ChurrouiClient, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypesUI)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	config := *cfg
	config.GroupVersion = &UISchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ChurrouiClient{restClient: client, ns: namespace}, nil
}

// ChurrouiClient ...
type ChurrouiClient struct {
	restClient rest.Interface
	ns         string
}

// Get ....
func (c *ChurrouiClient) Get(name string) (*uiv1alpha1.Churroui, error) {
	result := &uiv1alpha1.Churroui{}
	err := c.restClient.Get().
		Namespace(c.ns).Resource("churrouis").
		Name(name).Do(context.TODO()).Into(result)
	return result, err
}

// List ....
func (c *ChurrouiClient) List() (*uiv1alpha1.ChurrouiList, error) {
	result := &uiv1alpha1.ChurrouiList{}
	err := c.restClient.Get().
		Resource("churrouis").
		Do(context.TODO()).Into(result)
	return result, err
}

// Create ....
func (c *ChurrouiClient) Create(obj *uiv1alpha1.Churroui) (*uiv1alpha1.Churroui, error) {
	result := &uiv1alpha1.Churroui{}
	err := c.restClient.Post().
		Namespace(c.ns).Resource("churrouis").
		Body(obj).Do(context.TODO()).Into(result)
	return result, err
}

// Update ...
func (c *ChurrouiClient) Update(obj *uiv1alpha1.Churroui) (*uiv1alpha1.Churroui, error) {
	log.Info().Msg(fmt.Sprintf("updating churroui with %v\n", obj))
	result := &uiv1alpha1.Churroui{}
	err := c.restClient.Put().
		Namespace(c.ns).Resource("churrouis").Name(obj.Name).
		Body(obj).Do(context.TODO()).Into(result)
	if err != nil {
		log.Error().Stack().Err(err)
	}
	return result, err
}

// Delete ...
func (c *ChurrouiClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).Resource("churrouis").
		Name(name).Body(options).Do(context.TODO()).
		Error()
}
