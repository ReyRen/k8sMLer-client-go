package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"strconv"
	"time"
)

func resourceOperator(c *Client,
	kubeconfig string,
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

	// createk8s-client
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

			// respond to frontend get start msg
			c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
			c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = RECVSTART
			if c.addr != "" {
				c.hub.broadcast <- c
			}

			*realPvcName = Create_pvc(pvcClient, kindName, tmpString, labelName, &gracePeriodSeconds, caps)
			endStr, startStr := PraseTmpString(*realPvcName)
			for i := 0; i < nodeQuantity; i++ {
				_ = Create_service(svcClient, startStr+strconv.Itoa(i)+"-svc-"+endStr, labelName, &gracePeriodSeconds)
				Create_pod(podClient, startStr+strconv.Itoa(i)+"-pod-"+endStr, tmpString, labelName, int64(1), &gracePeriodSeconds, *realPvcName, i, nodeQuantity)
				for true {
					podPhase := Get_pod_status(podClient, kindName+strconv.Itoa(i)+"-pod-"+tmpString)
					if podPhase == apiv1.PodRunning {
						ip := get_10G_ips(podClient, kindName+strconv.Itoa(i)+"-pod-"+tmpString)
						c.hub.clients[*c.userIds].Head.ips += ip + ","
						break
					} else if podPhase == apiv1.PodPending {
						time.Sleep(time.Second * 3)
					} else if podPhase == apiv1.PodFailed {
						break
					} else if podPhase == apiv1.PodSucceeded {
						break
					} else if podPhase == apiv1.PodUnknown {
						break
					}
				}
			}
			exec_init_program(c, startStr+strconv.Itoa(nodeQuantity-1)+"-pod-"+endStr)
			//handle socket with the frontend
			//clientSocket(c, RESOURCECOMPLETE)
			log_back_to_frontend(c, kubeconfigName, nameSpace, c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes, &c.hub.clients[*c.userIds].Head.rm.realPvcName)
		case "service":
			_ = Create_service(svcClient, kindName, labelName, &gracePeriodSeconds)
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
				//Update_pod(podClient, kindName+strconv.Itoa(i)+"-pod-"+endStr)
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
		endStr, startStr := PraseTmpString(*realPvcName)
		fmt.Println("get pods log...")
		result := podClient.GetLogs(startStr+strconv.Itoa(nodeQuantity-1)+"-pod-"+endStr, &apiv1.PodLogOptions{
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
		//LogMonitor(podLogs)
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
