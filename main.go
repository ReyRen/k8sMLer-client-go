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
	var kubeconfig, namespaceName, serviceName, labelName string
	ParseArg(&kubeconfig, &namespaceName, &serviceName, &labelName)

	// create the client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfig)
	if err != nil {
		panic(err)
	}

	// create svc
	svcClient := clientset.CoreV1().Services(namespaceName)
	var service apiv1.Service
	ServiceCreate(&service, serviceName, labelName)
	fmt.Println("creating service...")
	result, err := svcClient.Create(context.TODO(), &service, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("create the service err : ", err)
	}
	fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
}
