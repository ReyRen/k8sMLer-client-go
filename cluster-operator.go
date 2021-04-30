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
	randomName string) {

	var nodeNum int
	var gpuNum int          // for each node
	if selectNodes != nil { // delete 的时候是nil所以小心空指针异常
		trimRsNum(selectNodes, &nodeNum, &gpuNum)
	} else {
		Error.Printf("[%d, %d]: selectNodes is nil\n", c.userIds.Uid, c.userIds.Tid)
	}

	getKubeconfigName(&kubeconfig) // fill up into the kubeconfig

	// create k8s-client
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

	switch operator {
	case "create":
		tmpString := GetRandomString(8) + "-" + strconv.Itoa(c.userIds.Uid) + "-" + strconv.Itoa(c.userIds.Tid)
		c.hub.clients[*c.userIds].Head.rm.RandomName = tmpString

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

			//*realPvcName = Create_pvc(pvcClient, kindName, tmpString, labelName, caps)
			//endStr, startStr := PraseTmpString(*realPvcName)

			var imageName string

			if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 7 {
				// 专有任务 -- 通过选择镜像列表
				imageName = REGISTRYSERVER + "/" + c.hub.clients[*c.userIds].Head.rm.Content.ImageName
			} else if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 6 {
				imageName = "172.18.29.81:8080/test-images/pytorch1.6_cuda10_horovod20.0_megatron_gpt2_gjx:02-02"
			} else if c.hub.clients[*c.userIds].Head.rm.Content.ToolBoxName == "mmdetection" {
				imageName = IMAGE_MMDECTION
			} else {
				imageName = IMAGE
			}
			/*if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 7 || c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 6 {
				// 专有任务 -- 通过选择镜像列表
				imageName = REGISTRYSERVER + "/" + c.hub.clients[*c.userIds].Head.rm.Content.ImageName
				if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 6 {
					imageName = "172.18.29.81:8080/test-images/pytorch1.6_cuda10_horovod20.0_megatron_gpt2_gjx:02-02"
				}
			} else {
				Create_pvc(pvcClient, kindName, tmpString, labelName, caps)
				if c.hub.clients[*c.userIds].Head.rm.Content.ToolBoxName == "mmdetection" {
					imageName = IMAGE_MMDECTION
				} else {
					imageName = IMAGE
				}
			}*/
			for i := 0; i < nodeNum; i++ {

				ip := getIpFromIppool()
				if ip == "" {
					Error.Printf("[%d, %d]:Assign speedup ip err\n", c.userIds.Uid, c.userIds.Tid)
					return
				}

				/*_ = Create_service(svcClient, kindName+strconv.Itoa(i)+"-svc-"+tmpString,
				labelName, &gracePeriodSeconds)*/
				Create_pod(podClient, kindName+strconv.Itoa(i)+"-pod-"+tmpString, ip, c.hub.clients[*c.userIds].Head.rm.RandomName,
					labelName, int64(gpuNum), &gracePeriodSeconds, i, nodeNum, imageName,
					(*selectNodes)[i].NodeNames,
					c.hub.clients[*c.userIds].Head.rm.Content.ModelType,
					c.hub.clients[*c.userIds].Head.rm.Content.ContinuousModelUrl,
					"/user/"+strconv.Itoa(c.userIds.Uid)+"/"+strconv.Itoa(c.userIds.Tid))
				for true {
					time.Sleep(time.Second * 3)
					podPhase := Get_pod_status(&(c.hub.clients[*c.userIds].Head.sm.Type), podClient, kindName+strconv.Itoa(i)+"-pod-"+c.hub.clients[*c.userIds].Head.rm.RandomName)
					if podPhase == apiv1.PodRunning {
						//c.hub.clients[*c.userIds].Head.rm.Type = 10
						//ip := get_10G_ips(podClient, kindName+strconv.Itoa(i)+"-pod-"+c.hub.clients[*c.userIds].Head.rm.RandomName)
						c.hub.clients[*c.userIds].Head.ips += ip + ","
						//Trace.Println(c.hub.clients[*c.userIds].Head.ips)
						break
					} else if podPhase == apiv1.PodPending {
						c.hub.broadcast <- c
						if c.hub.clients[*c.userIds].Head.sm.Type == INSUFFICIENTPENDING {
							return
						}
					} else if podPhase == apiv1.PodFailed {
						return
					} else if podPhase == apiv1.PodSucceeded {
						break
					} else if podPhase == apiv1.PodUnknown {
						return
					}
				}
			}
			//exec_init_program(c, startStr+strconv.Itoa(nodeQuantity-1)+"-pod-"+endStr)
			//handle socket with the frontend
			clientSocket(c, RESOURCECOMPLETE)

			c.hub.clients[*c.userIds].Head.ScheduleMap = POSTCREATE
			QUEUELIST = QUEUELIST[1:]

			/* ftp file name timestamp*/
			timestamp := time.Now().Unix()
			tm := time.Unix(timestamp, 0)
			timeStamp := tm.Format("20060102030405")

			fileName := strconv.Itoa(c.userIds.Uid) +
				"_" + strconv.Itoa(c.userIds.Tid) +
				"_" + timeStamp +
				"_log.txt"
			c.hub.clients[*c.userIds].Head.rm.FtpFileName = fileName
			/* ftp file name timestamp*/

			/* write back to .ippool file */
			writeIppoolToFile()

			log_back_to_frontend(c, kubeconfigName, nameSpace, kindName,
				c.hub.clients[*c.userIds].Head.rm.RandomName,
				nodeNum, gpuNum)
		case "service":
			_ = Create_service(svcClient, kindName, labelName, &gracePeriodSeconds)
		case "pvc":
			/*choose to use storageclass*/
			Create_pvc(pvcClient, kindName, tmpString, labelName, caps)
		default:
			Error.Println("resource is required[-o], only support pod,service")
		}
	case "delete":
		Trace.Printf("[%d, %d]:delete operation\n", c.userIds.Uid, c.userIds.Tid)
		switch resource {
		case "pod":
			//endStr, startStr := PraseTmpString(*realPvcName)
			for i := 0; i < nodeNum; i++ {
				Delete_pod(podClient, kindName+strconv.Itoa(i)+"-pod-"+c.hub.clients[*c.userIds].Head.rm.RandomName, labelName, &gracePeriodSeconds)
				//Delete_service(svcClient, kindName+strconv.Itoa(i)+"-svc-"+c.hub.clients[*c.userIds].Head.rm.RandomName, &gracePeriodSeconds)
			}
			/*used to write to udpate file*/
			go c.removeToUpdate()
			/* update ippool file*/
			writeIppool(c.hub.clients[*c.userIds].Head.ips)

			//Delete_pvc(pvcClient, kindName+"-pvc-"+c.hub.clients[*c.userIds].Head.rm.RandomName, labelName, &gracePeriodSeconds)
		case "service":
			Delete_service(svcClient, kindName, &gracePeriodSeconds)
		case "pvc":
			if c.hub.clients[*c.userIds].Head.rm.Content.ModelType != 7 {
				Delete_pvc(pvcClient, kindName, labelName, &gracePeriodSeconds)
			}
		default:
			Error.Println("resource is required[-o], only support pod,service")
		}
	case "log":
		//endStr, startStr := PraseTmpString(*realPvcName)
		Trace.Printf("[%d, %d]:get pods log\n", c.userIds.Uid, c.userIds.Tid)
		result := podClient.GetLogs(kindName+strconv.Itoa(nodeNum-1)+"-pod-"+randomName, &apiv1.PodLogOptions{
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
