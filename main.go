package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mheers/trino-gateway-operator/api"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: "trino.example.com", Version: "v1"}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&api.TrinoBackend{},
		&api.TrinoBackendList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

var (
	myScheme = runtime.NewScheme()
)

var (
	log = logrus.New()
)

var (
	gatewayAPIBaseURL = "https://query-trinogateway.cluster.local" // Replace with your Trino Gateway API URL
	gatewayUsername   = "trino-gateway"                            // Replace with your actual username
	gatewayPassword   = "test"                                     // Replace with your actual password
)

func init() {
	_ = AddToScheme(myScheme)
	_ = scheme.AddToScheme(myScheme)

	// Configure logrus
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// API_PASSWORD
	// API_URL
	// API_USER

	// replace gateway config with env vars if set
	if os.Getenv("API_URL") != "" {
		gatewayAPIBaseURL = os.Getenv("API_URL")
	}

	if os.Getenv("API_USER") != "" {
		gatewayUsername = os.Getenv("API_USER")
	}

	if os.Getenv("API_PASSWORD") != "" {
		gatewayPassword = os.Getenv("API_PASSWORD")
	}
}

func getClientConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// If in-cluster config fails, try kubeconfig file
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		// If KUBECONFIG is not set, use the default path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %v", err)
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	// Use the current context in kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %v", err)
	}

	return config, nil
}

func main() {
	log.Debug("Starting the Trino Backend operator")

	// Set up Kubernetes client configuration
	config, err := getClientConfig()
	if err != nil {
		log.Fatalf("Failed to get Kubernetes config: %v", err)
	}

	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}
	// Create a new RESTClient for our custom resources
	crdConfig := *config
	crdConfig.GroupVersion = &SchemeGroupVersion
	crdConfig.APIPath = "/apis"
	// crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(myScheme)
	crdConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	// Define the GroupVersionResource
	gvr := SchemeGroupVersion.WithResource("trinobackends")

	// Create a new queue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Create custom ListFunc and WatchFunc
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		return dynamicClient.Resource(gvr).Namespace(metav1.NamespaceAll).List(context.TODO(), options)
	}

	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		return dynamicClient.Resource(gvr).Namespace(metav1.NamespaceAll).Watch(context.TODO(), options)
	}

	// Create a new informer
	indexInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc:  listFunc,
			WatchFunc: watchFunc,
		},
		&unstructured.Unstructured{},
		0, // resync period
		cache.Indexers{},
	)

	indexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	// Start the controller
	stop := make(chan struct{})
	defer close(stop)
	go indexInformer.Run(stop)

	log.Info("Starting to process items from the queue")

	// Start the queue processing
	for {
		key, quit := queue.Get()
		if quit {
			log.Info("Received quit signal, shutting down")
			return
		}
		log.Debugf("Processing item: %s", key)
		err := processItem(key.(string), dynamicClient)
		if err != nil {
			log.Errorf("Error processing item: %v", err)
		}
		queue.Done(key)
	}

}

func createHTTPClientWithBasicAuth() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{
			// You can add more customizations here if needed
		},
	}
}

func addBasicAuth(req *http.Request) {
	req.SetBasicAuth(gatewayUsername, gatewayPassword)
}

func createOrUpdateBackend(backend api.TrinoBackend) error {
	log.Debugf("Creating or updating backend: %s", backend.Spec.Name)

	url := fmt.Sprintf("%s/entity?entityType=GATEWAY_BACKEND", gatewayAPIBaseURL)

	payload := map[string]interface{}{
		"name":         backend.Spec.Name,
		"proxyTo":      backend.Spec.ProxyTo,
		"active":       backend.Spec.Active,
		"routingGroup": backend.Spec.RoutingGroup,
	}
	if backend.Spec.ExternalUrl != "" {
		payload["externalUrl"] = backend.Spec.ExternalUrl
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal backend payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	addBasicAuth(req)

	client := createHTTPClientWithBasicAuth()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create/update backend, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TODO: not working yet. Even curl does not work: curl -X POST -d "example-trinobackend" "https://query-trinogateway.cluster.local/gateway/backend/modify/delete" -vvv -u trino-gateway:test; it returns a 200 OK but does not delete the backend
func deleteBackend(name string) error {
	log.Debugf("Deleting backend: %s", name)

	url := fmt.Sprintf("%s/gateway/backend/modify/delete", gatewayAPIBaseURL)

	req, err := http.NewRequest("POST", url, strings.NewReader(name))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addBasicAuth(req)

	client := createHTTPClientWithBasicAuth()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete backend, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func processItem(key string, dynamicClient dynamic.Interface) error {
	log.Debugf("Processing item: %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Get the TrinoBackend resource
	gvr := SchemeGroupVersion.WithResource("trinobackends")
	unstructured, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// The resource was not found, which likely means it was deleted
			log.Infof("TrinoBackend %s/%s not found, assuming it's been deleted", namespace, name)
			return deleteBackend(name)
		}
		return fmt.Errorf("failed to get TrinoBackend: %v", err)
	}

	var backend api.TrinoBackend
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &backend)
	if err != nil {
		return fmt.Errorf("failed to convert unstructured to TrinoBackend: %v", err)
	}

	// Process the TrinoBackend based on its state
	if backend.ObjectMeta.DeletionTimestamp != nil {
		// The resource is being deleted
		log.Debugf("TrinoBackend %s/%s is being deleted", namespace, name)
		return deleteBackend(backend.Spec.Name)
	}

	// The resource is being created or updated
	log.Debugf("Creating or updating TrinoBackend %s/%s", namespace, name)
	return createOrUpdateBackend(backend)
}
