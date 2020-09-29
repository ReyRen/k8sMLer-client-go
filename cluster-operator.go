package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"strconv"
)

func resourceOperator(kubeconfig string,
	operator string,
	resource string,
	namespaceName string,
	kindName string,
	labelName string,
	caps string,
	nodeQuantity int,
	realPvcName *string) { // realPvcName used to get created random rs group name

	//gpuQuantity int64) {
	/*var kubeconfig, operator, resource, namespaceName, kindName, labelName, caps string
	var gpuQuantity int64

	ParseArg(&kubeconfig,
		&operator,
		&resource,
		&namespaceName,
		&kindName,
		&labelName,
		&gpuQuantity,
		&caps,
	)*/

	getKubeconfigName(&kubeconfig) // fill up into the kubeconfig

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
	/* choose to use the StoragClass*/
	//pvClient := clientset.CoreV1().PersistentVolumes()

	// delete graceful
	gracePeriodSeconds := int64(0) // delete immediately
	// genereate the same randomString at the same time
	tmpString := GetRandomString(15)

	switch operator {
	case "create":
		fmt.Println("create operation...")
		switch resource {
		case "pod":
			*realPvcName = Create_pvc(pvcClient, kindName, tmpString, labelName, &gracePeriodSeconds, caps)
			for i := 0; i < nodeQuantity; i++ {
				_ = Create_service(svcClient, kindName+strconv.Itoa(i), tmpString, labelName, &gracePeriodSeconds)
				Create_pod(podClient, kindName+strconv.Itoa(i), tmpString, labelName, int64(1), &gracePeriodSeconds, *realPvcName)
			}
		case "service":
			_ = Create_service(svcClient, kindName, tmpString, labelName, &gracePeriodSeconds)
		case "pvc":
			/*choose to use storageclass*/
			_ = Create_pvc(pvcClient, kindName, tmpString, labelName, &gracePeriodSeconds, caps)
		default:
			log.Fatal("resource is required[-o], only support pod,service")
		}
	case "delete":
		fmt.Println("delete operation...")
		switch resource {
		case "pod":
			endStr, startStr := PraseTmpString(*realPvcName)
			for i := 0; i < nodeQuantity; i++ {
				Delete_pod(podClient, kindName+strconv.Itoa(i)+"-pod-"+endStr, labelName, &gracePeriodSeconds)
				Delete_service(svcClient, startStr+strconv.Itoa(i)+"-svc-"+endStr, &gracePeriodSeconds)
			}
			Delete_pvc(pvcClient, startStr+"-pvc-"+endStr, labelName, &gracePeriodSeconds)
		case "service":
			Delete_service(svcClient, kindName, &gracePeriodSeconds)
		case "pvc":
			Delete_pvc(pvcClient, kindName, labelName, &gracePeriodSeconds)
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
			List_pod(podClient, labelName)
		case "service":
			List_service(svcClient, labelName)
		case "pvc":
			List_pvc(pvcClient, labelName)
		default:
			log.Fatal("resource is required[-o], only support pod,service")
		}
	}
}
