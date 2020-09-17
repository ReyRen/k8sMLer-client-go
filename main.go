package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"

	//"k8s.io/client-go/util/retry"
	"log"
	//"os"
	//"text/template/parse"

	//appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	var kubeconfig, operator, resource, namespaceName, kindName, labelName string
	var gpuQuantity int64

	ParseArg(&kubeconfig, &operator, &resource, &namespaceName, &kindName, &labelName, &gpuQuantity)

	// create k8s-client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfig)
	if err != nil {
		panic(err)
	}

	// define resource
	svcClient := clientset.CoreV1().Services(namespaceName)
	podClient := clientset.CoreV1().Pods(namespaceName)

	switch operator {
	case "create":
		fmt.Println("create operation...")
		switch resource {
		case "pod":
			fmt.Println("creating pod...")
			var pod apiv1.Pod
			PodReady(&pod, kindName, labelName, gpuQuantity)
			result, err := podClient.Create(context.TODO(), &pod, metav1.CreateOptions{})
			if err != nil {
				log.Fatalln("create the pod err : ", err)
			}
			fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
		case "service":
			var service apiv1.Service
			ServiceReady(&service, kindName, labelName)
			fmt.Println("creating service...")
			result, err := svcClient.Create(context.TODO(), &service, metav1.CreateOptions{})
			if err != nil {
				log.Fatalln("create the service err : ", err)
			}
			fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
		default:
			log.Fatal("resource is required[-o], only support pod,service")
		}
	case "delete":
		fmt.Println("delete operation...")
		switch resource {
		case "pod":
			fmt.Println("delete pod...")
		case "service":
			fmt.Println("delete service...")
			deletePolicy := metav1.DeletePropagationForeground
			//gracePeriodSeconds := new(int64) // You have a pointer variable which after declaration will be nil
			// if you want to set the pointed value, it must point to something
			// Attempting to dereference a nil pointer is a runtime panic
			//gracePeriodSeconds = new(int64)
			gracePeriodSeconds := int64(0) // delete immediately
			if err := svcClient.Delete(context.TODO(), kindName, metav1.DeleteOptions{
				GracePeriodSeconds: &gracePeriodSeconds,
				PropagationPolicy:  &deletePolicy,
			}); err != nil {
				log.Fatalln("delete svc err: ", err)
			}
			fmt.Printf("deleted service %s\n", kindName)
		default:
			log.Fatal("resource is required[-o], only support pod,service")
		}
	default:
		fmt.Println("list operation...")
		switch resource {
		case "pod":
			fmt.Println("list pod...")
		case "service":
			fmt.Println("list service...")
			list, err := svcClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
			if err != nil {
				log.Fatalln("list svc err: ", err)
			}
			for _, s := range list.Items {
				fmt.Printf(" * [%s] svc in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
			}
		default:
			log.Fatal("resource is required[-o], only support pod,service")
		}
	}
}
