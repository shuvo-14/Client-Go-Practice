package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"log"
	"os"
	"time"
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
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("error %s,creating clientset", err.Error())
	}
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "bookapiserver",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "bookapiserver",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "shuvo14/api-server",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
	prompt()

	// service
	serviceClients := clientset.CoreV1().Services(apiv1.NamespaceDefault)
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bookapiserver",
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": "bookapiserver",
			},
			Ports: []apiv1.ServicePort{
				{
					Protocol:   apiv1.ProtocolTCP,
					Port:       3200,
					TargetPort: intstr.FromInt32(8080),
				},
			},
		},
	}
	/// Creating service
	fmt.Println("Creating Service...")
	res, err := serviceClients.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created service %q.\n", res.GetObjectMeta().GetName())
	time.Sleep(1 * time.Minute)
	// Deleting service
	fmt.Println("Deleting Service...")
	prompt()
	err = serviceClients.Delete(context.TODO(), res.GetObjectMeta().GetName(), metav1.DeleteOptions{})
	fmt.Printf("Deleted Service %q.\n", res.GetObjectMeta().GetName())

	fmt.Println("Analyzing our new deployment\n")

	ctx := context.Background()
	pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("error %s, while listening all the pods from default namespace\n", err.Error())
	}
	fmt.Println("Pods from default namespace")
	for _, pod := range pods.Items {
		fmt.Printf("%s\n", pod.Name)
	}

	prompt()
	fmt.Println("Updating deployment...")

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := deploymentsClient.Get(context.Background(), "demo-deployment", metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		result.Spec.Replicas = int32Ptr(1)
		result.Spec.Template.Spec.Containers[0].Image = "nginx:1.13"
		_, err = deploymentsClient.Update(context.Background(), result, metav1.UpdateOptions{})
		return err
	})

	if retryErr != nil {
		log.Fatal(err)
	}

	fmt.Println("Updated deployment...")
	prompt()
	fmt.Printf("Listening deployments in namespace %s: \n", apiv1.NamespaceDefault)
	list, err := deploymentsClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, deploy := range list.Items {
		fmt.Printf("Deployment Name: %s and have %v replicas\n", deploy.Name, *deploy.Spec.Replicas)
	}

	prompt()

	fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.Background(), "demo-deployment", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deleted deployment...")
}
func int32Ptr(i int32) *int32 { return &i }
func prompt() {
	fmt.Printf("->Press Return key to continue")
	scannar := bufio.NewScanner(os.Stdin)
	for scannar.Scan() {
		break
	}
	if err := scannar.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}
