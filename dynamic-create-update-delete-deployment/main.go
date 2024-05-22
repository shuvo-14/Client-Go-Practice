package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
	"log"
	"os"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/home/appscode-pc/.kube/config", "location")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	fmt.Println(kubeconfig)
	if err != nil {
		fmt.Printf("error %s, creating config files", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("error %s, creating in cluster config files", err.Error())

		}
	}
	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("error %s, creating clientset", err.Error())
	}
	deploymentsRes := schema.GroupVersionResource{
		Group: "apps", Version: "v1", Resource: "deployments",
	}
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "demo-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "demo",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "demo",
						},
					},
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "web",
								"image": "shuvo14/api-server",
								"ports": []map[string]interface{}{
									{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	fmt.Println("Creating deployment...")
	result, err := clientset.Resource(deploymentsRes).Namespace(apiv1.NamespaceDefault).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetName())
	prompt()
	fmt.Println("Updating deployment...")

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := clientset.Resource(deploymentsRes).Namespace(apiv1.NamespaceDefault).Get(context.Background(), "demo-deployment", metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		err = unstructured.SetNestedField(result.Object, int64(1), "spec", "replicas")
		if err != nil {
			log.Fatal(err)
		}

		containers, found, err := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "containers")
		if err != nil || !found || containers == nil {
			log.Fatal(err)
		}
		if err := unstructured.SetNestedField(containers[0].(map[string]interface{}), "nginx:1.13", "image"); err != nil {
			panic(err)
		}
		if err := unstructured.SetNestedField(result.Object, containers, "spec", "template", "spec", "containers"); err != nil {
			panic(err)
		}

		_, err = clientset.Resource(deploymentsRes).Namespace(apiv1.NamespaceDefault).Update(context.Background(), result, metav1.UpdateOptions{})
		return err
	})

	if retryErr != nil {
		log.Fatal(err)
	}

	fmt.Println("Updated deployment...")
	prompt()
	fmt.Printf("Listing deployments in namespace %s: \n", apiv1.NamespaceDefault)
	list, err := clientset.Resource(deploymentsRes).Namespace(apiv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, deploy := range list.Items {
		replicas, found, err := unstructured.NestedInt64(deploy.Object, "spec", "replicas")
		if err != nil || !found {
			fmt.Printf("Replicas not found for deployment %s: error=%s", deploy.GetName(), err)
			continue
		}
		fmt.Printf("Deployment Name: %s and have %v replicas\n", deploy.GetName(), replicas)
	}

	prompt()

	fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := clientset.Resource(deploymentsRes).Namespace(apiv1.NamespaceDefault).Delete(context.Background(), "demo-deployment", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deleted deployment...")

}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}
