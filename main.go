package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
)

func main() {
	var kubeconfig, operator, resource, namespaceName, kindName, labelName, cap string
	var gpuQuantity int64

	ParseArg(&kubeconfig, &operator, &resource, &namespaceName, &kindName, &labelName, &gpuQuantity, &cap)

	// create k8s-client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfig)
	if err != nil {
		panic(err)
	}

	// define resource
	svcClient := clientset.CoreV1().Services(namespaceName)
	podClient := clientset.CoreV1().Pods(namespaceName)
	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespaceName)

	// delete graceful
	//gracePeriodSeconds := new(int64) // You have a pointer variable which after declaration will be nil
	// if you want to set the pointed value, it must point to something
	// Attempting to dereference a nil pointer is a runtime panic
	//gracePeriodSeconds = new(int64)
	gracePeriodSeconds := int64(0) // delete immediately

	switch operator {
	case "create":
		fmt.Println("create operation...")
		switch resource {
		case "pod":
			var pod apiv1.Pod
			PodReady(&pod, kindName, labelName, gpuQuantity, &gracePeriodSeconds)
			fmt.Println("creating pod...")
			result, err := podClient.Create(context.TODO(), &pod, metav1.CreateOptions{})
			if err != nil {
				log.Fatalln("create the pod err : ", err)
			}
			fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
		case "service":
			var service apiv1.Service
			ServiceReady(&service, kindName, labelName, &gracePeriodSeconds)
			fmt.Println("creating service...")
			result, err := svcClient.Create(context.TODO(), &service, metav1.CreateOptions{})
			if err != nil {
				log.Fatalln("create the service err : ", err)
			}
			fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
		case "pvc":
			var pvcs apiv1.PersistentVolumeClaim
			PvcReady(&pvcs, kindName, labelName, &gracePeriodSeconds, cap)
			fmt.Println("creating pvc...")
			result, err := pvcClient.Create(context.TODO(), &pvcs, metav1.CreateOptions{})
			if err != nil {
				log.Fatalln("create the pvc err : ", err)
			}
			fmt.Printf("Created persistentvolumeclaim %q.\n", result.GetObjectMeta().GetName())
		default:
			log.Fatal("resource is required[-o], only support pod,service")
		}
	case "delete":
		fmt.Println("delete operation...")
		switch resource {
		case "pod":
			deletePolicy := metav1.DeletePropagationForeground
			if kindName != "" {
				fmt.Println("delete pod...")
				if err := podClient.Delete(context.TODO(), kindName, metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriodSeconds,
					PropagationPolicy:  &deletePolicy,
				}); err != nil {
					log.Fatalln("delete pod err:", err)
				}
				fmt.Printf("deleted pod %s\n", kindName)
			} else {
				fmt.Println("delete pods...")
				if err := podClient.DeleteCollection(context.TODO(), metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriodSeconds,
					PropagationPolicy:  &deletePolicy,
				}, metav1.ListOptions{
					TypeMeta:      metav1.TypeMeta{},
					LabelSelector: labelName,
					FieldSelector: "",
					Watch:         true,
				}); err != nil {
					log.Fatalln("delete pods err:", err)
				}
				fmt.Printf("delete all pods under label: %s\n", labelName)
			}
		case "service":
			fmt.Println("delete service...")
			deletePolicy := metav1.DeletePropagationForeground
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
	case "log":
		fmt.Println("get pods log...")
		result := podClient.GetLogs(kindName, &apiv1.PodLogOptions{
			Container:  "",
			Follow:     true,
			Previous:   false,
			Timestamps: true, // timestamps
		})
		podLogs, err := result.Stream(context.TODO())
		if err != nil {
			log.Fatalln("podLogs stream err : ", err)
		}
		defer podLogs.Close()
		LogMonitor(podLogs)
	default:
		fmt.Println("list operation...")
		switch resource {
		case "pod":
			fmt.Println("list pod...")
			list, err := podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
			if err != nil {
				log.Fatalln("list pod err: ", err)
			}
			for _, s := range list.Items {
				fmt.Printf(" * [%s] pod in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
			}
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
