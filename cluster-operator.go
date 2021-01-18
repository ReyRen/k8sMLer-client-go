package main

import (
	"context"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
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
	selectNodes *[]selectNodes,
	realPvcName *string) { // realPvcName used to get created random rs group name

	var nodeNum int
	var gpuNum int // for each node
	trimRsNum(selectNodes, &nodeNum, &gpuNum)

	getKubeconfigName(&kubeconfig) // fill up into the kubeconfig

	// createk8s-client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfig)
	if err != nil {
		Error.Printf("[%d, %d]:createClient err: %s")
		panic(err)
	}

	// define resource
	svcClient := clientset.CoreV1().Services(namespaceName)
	podClient := clientset.CoreV1().Pods(namespaceName)
	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespaceName)
	/* choose to use the StoragClass*/

	// delete graceful
	gracePeriodSeconds := int64(0) // delete immediately
	// genereate the same randomString at the same time
	tmpString := GetRandomString(15)

	switch operator {
	case "create":
		Trace.Printf("[%d, %d]:create operation\n", c.userIds.Uid, c.userIds.Tid)
		switch resource {
		case "pod":

			// respond to frontend get start msg
			/*c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
			c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = RECVSTART*/
			c.hub.clients[*c.userIds].Head.sm.Type = RECVSTART
			if c.addr != "" {
				c.hub.broadcast <- c
			}

			*realPvcName = Create_pvc(pvcClient, kindName, tmpString, labelName, caps)
			endStr, startStr := PraseTmpString(*realPvcName)

			var imageName string
			if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 7 {
				// 专有任务 -- 通过选择镜像列表
				imageName = REGISTRYSERVER + "/" + c.hub.clients[*c.userIds].Head.rm.Content.ImageName
			} else {
				if c.hub.clients[*c.userIds].Head.rm.Content.ToolBoxName == "mmdection" {
					imageName = IMAGE_MMDECTION
				} else {
					imageName = IMAGE
				}
			}
			for i := 0; i < nodeNum; i++ {
				_ = Create_service(svcClient, startStr+strconv.Itoa(i)+"-svc-"+endStr,
					labelName, &gracePeriodSeconds)
				Create_pod(podClient, startStr+strconv.Itoa(i)+"-pod-"+endStr, tmpString,
					labelName, int64(gpuNum), &gracePeriodSeconds, *realPvcName, i,
					nodeNum, imageName, (*selectNodes)[i].NodeNames)
				for true {
					time.Sleep(time.Second * 3)
					podPhase := Get_pod_status(&(c.hub.clients[*c.userIds].Head.sm.Type), podClient, kindName+strconv.Itoa(i)+"-pod-"+tmpString)
					if podPhase == apiv1.PodRunning {
						//c.hub.clients[*c.userIds].Head.rm.Type = 10
						ip := get_10G_ips(podClient, kindName+strconv.Itoa(i)+"-pod-"+tmpString)
						c.hub.clients[*c.userIds].Head.ips += ip + ","
						break
					} else if podPhase == apiv1.PodPending {
						c.hub.broadcast <- c
						if c.hub.clients[*c.userIds].Head.sm.Type == INSUFFICIENTPENDING {
							break
						}
					} else if podPhase == apiv1.PodFailed {
						break
					} else if podPhase == apiv1.PodSucceeded {
						break
					} else if podPhase == apiv1.PodUnknown {
						break
					}
				}
			}
			//exec_init_program(c, startStr+strconv.Itoa(nodeQuantity-1)+"-pod-"+endStr)
			//handle socket with the frontend
			clientSocket(c, RESOURCECOMPLETE)
			log_back_to_frontend(c, kubeconfigName, nameSpace,
				&c.hub.clients[*c.userIds].Head.rm.realPvcName,
				nodeNum, gpuNum)
		case "service":
			_ = Create_service(svcClient, kindName, labelName, &gracePeriodSeconds)
		case "pvc":
			/*choose to use storageclass*/
			_ = Create_pvc(pvcClient, kindName, tmpString, labelName, caps)
		default:
			Error.Println("resource is required[-o], only support pod,service")
		}
	case "delete":
		Trace.Printf("[%d, %d]:delete operation\n", c.userIds.Uid, c.userIds.Tid)
		switch resource {
		case "pod":
			endStr, startStr := PraseTmpString(*realPvcName)
			for i := 0; i < nodeNum; i++ {
				Delete_pod(podClient, kindName+strconv.Itoa(i)+"-pod-"+endStr, labelName, &gracePeriodSeconds)
				Delete_service(svcClient, startStr+strconv.Itoa(i)+"-svc-"+endStr, &gracePeriodSeconds)
			}
			Delete_pvc(pvcClient, startStr+"-pvc-"+endStr, labelName, &gracePeriodSeconds)
		case "service":
			Delete_service(svcClient, kindName, &gracePeriodSeconds)
		case "pvc":
			Delete_pvc(pvcClient, kindName, labelName, &gracePeriodSeconds)
		default:
			Error.Println("resource is required[-o], only support pod,service")
		}
	case "log":
		endStr, startStr := PraseTmpString(*realPvcName)
		Trace.Printf("[%d, %d]:get pods log\n", c.userIds.Uid, c.userIds.Tid)
		result := podClient.GetLogs(startStr+strconv.Itoa(nodeNum-1)+"-pod-"+endStr, &apiv1.PodLogOptions{
			Container:  "",
			Follow:     true,
			Previous:   false,
			Timestamps: true, // timestamps
		})
		podLogs, err := result.Stream(context.TODO())
		if err != nil {
			Error.Println("podLogs stream err : ", err)
		}
		defer podLogs.Close()
		//LogMonitor(podLogs)
	default:
		Trace.Printf("[%d, %d]:list operation\n", c.userIds.Uid, c.userIds.Tid)
		switch resource {
		case "pod":
			List_pod(podClient, labelName)
		case "service":
			List_service(svcClient, labelName)
		case "pvc":
			List_pvc(pvcClient, labelName)
		default:
			Error.Println("resource is required[-o], only support pod,service")
		}
	}
}
