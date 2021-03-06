package main

import (
	"context"
	"encoding/json"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"strconv"
	"strings"
)

func PodReady(pods *apiv1.Pod, podName string, tmpString string,
	labelName string, gpuQuantity int64, gracePeriodSeconds *int64,
	currentI int, totalI int, imageName string, bindName string) {

	// assemble a container name
	containName := podName + "-container-" + tmpString

	// multus-cni for different interface in pods
	//multus := make(map[string]string)
	//multus["k8s.v1.cni.cncf.io/networks"] = "macvlan-conf"

	// volumeMount
	mountPath := MOUNTPATH
	mountName := podName + "-mount-" + tmpString

	// get the execute args
	var args []string
	if currentI == totalI-1 {
		// last one pod
		//args = []string{INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + ";python " + START_IN_POD + END_TAIL}
		args = []string{INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + ";python " + START_IN_POD}
	} else {
		args = []string{INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + END_TAIL}
		//args = []string{INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL}
	}

	// assemble a resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	var resourceQuantity resource.Quantity
	resourceQuantity.Set(gpuQuantity)
	resourceLimit["nvidia.com/gpu"] = resourceQuantity

	*pods = apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: map[string]string{labelName: labelName},
			//Annotations: multus, // need for multus-cni
		},
		Spec: apiv1.PodSpec{
			Volumes: []apiv1.Volume{
				{
					Name: mountName,
					VolumeSource: apiv1.VolumeSource{
						PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "gpu-pvc-" + tmpString,
						},
					},
				},
			},
			NodeSelector: map[string]string{"kubernetes.io/hostname": bindName},
			Containers: []apiv1.Container{
				{
					Name:       containName,
					Image:      imageName,
					Command:    []string{"/bin/sh", "-c"},
					Args:       args,
					WorkingDir: "",
					Ports:      []apiv1.ContainerPort{},
					Env:        nil,
					Resources: apiv1.ResourceRequirements{
						Limits:   resourceLimit,
						Requests: nil,
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      mountName,
							MountPath: mountPath,
						},
					},
					TerminationMessagePolicy: apiv1.TerminationMessageFallbackToLogsOnError,
					/*
						TerminationMessageFallbackToLogsOnError will read the most recent contents
						of the container logs for the container status message when the container
						exits with an error and the termination Message Path has no contents.
					*/
				},
			},
			RestartPolicy:                 "",
			TerminationGracePeriodSeconds: gracePeriodSeconds,
			//NodeSelector: map[string]string{labelName: labelName},
			NodeName:          "", // auto
			Hostname:          "",
			Affinity:          nil,
			PriorityClassName: "",
			Priority:          nil,
		},
		Status: apiv1.PodStatus{},
	}
}

func PodReady2(pods *apiv1.Pod, podName string, ip string, tmpString string,
	labelName string, gpuQuantity int64, gracePeriodSeconds *int64,
	currentI int, totalI int, imageName string, bindName string, continueModelURL string, selfModelUrl string) {

	//headDir := "/home/ftper/ftp"
	// continueModelURL = /ftp/user/11/166/result
	// modelName = erhgreh.rar

	// assemble a container name
	containName := podName + "-container-" + tmpString
	changehostnameIP := "echo \"" + ip + " " + podName + "\" > /etc/hosts;"

	// multus-cni for different interface in pods
	multus := make(map[string]string)
	ip = ip + "/24"
	valueString := "[" + "{ " + "\"name\": \"macvlan-conf\",\"ips\": [ \"" + ip + "\" ]}]"
	multus["k8s.v1.cni.cncf.io/networks"] = valueString

	// get the execute args
	var args []string
	if currentI == totalI-1 {
		//args = []string{INIT_TAIL + ";sleep 3000;python /storage-root/scripts/start.py" + END_TAIL}
		// last one pod
		args = []string{changehostnameIP + INIT_TAIL + ";sleep 3;python /storage-root/scripts/start.py" + END_TAIL}
	} else {
		args = []string{changehostnameIP + INIT_TAIL + END_TAIL}
	}

	// assemble a resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	var resourceQuantityGPU resource.Quantity
	/*var resourceQuantityMem resource.Quantity
	var resourceQuantityCPU resource.Quantity*/
	resourceQuantityGPU.Set(gpuQuantity)
	/*resourceQuantityMem.Set(gpuQuantity * 20 * 1024 * 1024 * 1024)
	resourceQuantityCPU.Set(gpuQuantity * 4)*/
	resourceLimit["nvidia.com/gpu"] = resourceQuantityGPU
	/*resourceLimit[apiv1.ResourceMemory] = resourceQuantityMem
	resourceLimit[apiv1.ResourceCPU] = resourceQuantityCPU*/

	if continueModelURL == "" {
		// 非续训
		*pods = apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        podName,
				Labels:      map[string]string{labelName: labelName},
				Annotations: multus, // need for multus-cni
			},
			Spec: apiv1.PodSpec{
				Volumes: []apiv1.Volume{
					{
						Name: "dshm",
						VolumeSource: apiv1.VolumeSource{
							EmptyDir: &apiv1.EmptyDirVolumeSource{
								Medium:    apiv1.StorageMediumMemory,
								SizeLimit: nil,
							},
						},
					},
					{
						Name: "datasets",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + "/datasets/",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + "/datasets/",
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "models",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + selfModelUrl + "/result/",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + selfModelUrl + "/result/",
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "scripts",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + "/script/",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + "/script/",
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "tblog",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + selfModelUrl + "/TensorBoardLog",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + selfModelUrl + "/TensorBoardLog",
								ReadOnly: false,
							},*/
						},
					},
				},
				NodeSelector: map[string]string{"kubernetes.io/hostname": bindName},
				Containers: []apiv1.Container{
					{
						Name:       containName,
						Image:      imageName,
						Command:    []string{"/bin/sh", "-c"},
						Args:       args,
						WorkingDir: "",
						Ports:      []apiv1.ContainerPort{},
						Env:        nil,
						Resources: apiv1.ResourceRequirements{
							Limits:   resourceLimit,
							Requests: nil,
						},
						VolumeMounts: []apiv1.VolumeMount{
							{
								Name:      "dshm",
								MountPath: "/dev/shm",
							},
							{
								Name:      "datasets",
								MountPath: "/storage-root/datasets",
							},
							{
								Name:      "models",
								MountPath: "/storage-root/models",
							},
							{
								Name:      "scripts",
								MountPath: "/storage-root/scripts",
							},
							{
								Name:      "tblog",
								MountPath: "/storage-root/TensorBoardLog",
							},
						},
						TerminationMessagePolicy: apiv1.TerminationMessageFallbackToLogsOnError,
						/*
							TerminationMessageFallbackToLogsOnError will read the most recent contents
							of the container logs for the container status message when the container
							exits with an error and the termination Message Path has no contents.
						*/
					},
				},
				RestartPolicy:                 "",
				TerminationGracePeriodSeconds: gracePeriodSeconds,
				//NodeSelector: map[string]string{labelName: labelName},
				NodeName:          "", // auto
				Hostname:          "",
				Affinity:          nil,
				PriorityClassName: "",
				Priority:          nil,
				//HostNetwork:       true,
			},
			Status: apiv1.PodStatus{},
		}
	} else {
		*pods = apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        podName,
				Labels:      map[string]string{labelName: labelName},
				Annotations: multus, // need for multus-cni
			},
			Spec: apiv1.PodSpec{
				Volumes: []apiv1.Volume{
					{
						Name: "dshm",
						VolumeSource: apiv1.VolumeSource{
							EmptyDir: &apiv1.EmptyDirVolumeSource{
								Medium:    apiv1.StorageMediumMemory,
								SizeLimit: nil,
							},
						},
					},
					{
						Name: "datasets",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + "/datasets/",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + "/datasets/",
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "models",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + selfModelUrl + "/result/",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + selfModelUrl + "/result/",
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "models-parent",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + continueModelURL,
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + continueModelURL,
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "scripts",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + "/script/",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + "/script/",
								ReadOnly: false,
							},*/
						},
					},
					{
						Name: "tblog",
						VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/storage-ftp-data" + selfModelUrl + "/TensorBoardLog",
								Type: nil,
							},
							/*NFS: &apiv1.NFSVolumeSource{
								Server:   DATA_WEB_SERVER,
								Path:     headDir + selfModelUrl + "/TensorBoardLog",
								ReadOnly: false,
							},*/
						},
					},
				},
				NodeSelector: map[string]string{"kubernetes.io/hostname": bindName},
				Containers: []apiv1.Container{
					{
						Name:       containName,
						Image:      imageName,
						Command:    []string{"/bin/sh", "-c"},
						Args:       args,
						WorkingDir: "",
						Ports:      []apiv1.ContainerPort{},
						Env:        nil,
						Resources: apiv1.ResourceRequirements{
							Limits:   resourceLimit,
							Requests: nil,
						},
						VolumeMounts: []apiv1.VolumeMount{
							{
								Name:      "dshm",
								MountPath: "/dev/shm",
							},
							{
								Name:      "datasets",
								MountPath: "/storage-root/datasets",
							},
							{
								Name:      "models",
								MountPath: "/storage-root/models",
							},
							{
								Name:      "models-parent",
								MountPath: "/storage-root/models-parent",
							},
							{
								Name:      "scripts",
								MountPath: "/storage-root/scripts",
							},
							{
								Name:      "tblog",
								MountPath: "/storage-root/TensorBoardLog",
							},
						},
						TerminationMessagePolicy: apiv1.TerminationMessageFallbackToLogsOnError,
						/*
							TerminationMessageFallbackToLogsOnError will read the most recent contents
							of the container logs for the container status message when the container
							exits with an error and the termination Message Path has no contents.
						*/
					},
				},
				RestartPolicy:                 "",
				TerminationGracePeriodSeconds: gracePeriodSeconds,
				//NodeSelector: map[string]string{labelName: labelName},
				NodeName:          "", // auto
				Hostname:          "",
				Affinity:          nil,
				PriorityClassName: "",
				Priority:          nil,
				//HostNetwork:       true,
			},
			Status: apiv1.PodStatus{},
		}
	}
}

func Create_pod(podClient v1.PodInterface,
	podName string,
	ip string,
	tmpString string,
	labelName string,
	gpuQuantity int64,
	gracePeriodSeconds *int64,
	currentI int,
	totalI int,
	imageName string,
	bindName string,
	modelType int,
	continuousModelUrl string,
	selfModelUrl string) {
	var pod apiv1.Pod
	/*
	   TODO: 功能合并
	*/
	PodReady2(&pod, podName, ip, tmpString, labelName, gpuQuantity, gracePeriodSeconds,
		currentI, totalI, imageName, bindName, continuousModelUrl, selfModelUrl)
	/*if modelType == 7 || modelType == 6 {
		PodReady2(&pod, podName, tmpString, labelName, gpuQuantity, gracePeriodSeconds,
			currentI, totalI, imageName, bindName, continuousModelUrl, selfModelUrl)
	} else {
		PodReady(&pod, podName, tmpString, labelName, gpuQuantity, gracePeriodSeconds,
			currentI, totalI, imageName, bindName)
	}*/
	_, err := podClient.Create(context.TODO(), &pod, metav1.CreateOptions{})
	if err != nil {
		Error.Println("create the pod err : ", err)
	}
	Trace.Printf("created %s\n", podName)
}

func List_pod(podClient v1.PodInterface, labelName string) {
	fmt.Println("list pod...")
	list, err := podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
	if err != nil {
		Error.Println("list pod err: ", err)
	}
	for _, s := range list.Items {
		Trace.Printf(" * [%s] pod in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
	}
}

func Delete_pod(podClient v1.PodInterface, podName string, labelName string, gracePeriodSeconds *int64) {
	deletePolicy := metav1.DeletePropagationForeground
	if podName != "" {
		if err := podClient.Delete(context.TODO(), podName, metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}); err != nil {
			Error.Println("delete pod err:", err)
		}
		Trace.Println("deleted ", podName)
	} else {
		if err := podClient.DeleteCollection(context.TODO(), metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}, metav1.ListOptions{
			TypeMeta:      metav1.TypeMeta{},
			LabelSelector: labelName,
			FieldSelector: "",
			Watch:         true,
		}); err != nil {
			Error.Println("delete pods err:", err)
		}
		Trace.Printf("delete all pods under label: %s\n", labelName)
	}
}

func Get_pod_status(statusType *int, podClient v1.PodInterface, podName string) apiv1.PodPhase {
	podv1, _ := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
	podconditions := podv1.Status.Conditions
	/*for i := 0; i < len(podconditions); i++ {
	fmt.Printf("%d..%d...%d..%d..%d..%d..\n", i, i, i, i, i, i)
	fmt.Printf("lens: %d\n", len(podconditions))
	fmt.Printf("v.Type: %s\n", podconditions[i].Type)
	fmt.Printf("v.Status: %s\n", podconditions[i].Status)
	fmt.Printf("v.Reason: %s\n", podconditions[i].Reason)
	fmt.Printf("v.Message: %s\n", podconditions[i].Message)*/
	/*
		0..0...0..0..0..0..
		lens: 1
		v.Type: PodScheduled
		v.Status: True
		v.Reason:
		v.Message:
		0..0...0..0..0..0..
		lens: 4
		v.Type: Initialized
		v.Status: True
		v.Reason:
		v.Message:
		1..1...1..1..1..1..
		lens: 4
		v.Type: Ready
		v.Status: False
		v.Reason: ContainersNotReady
		v.Message: containers with unready status: [gpu0-pod-eqp78m0kgaq0m57-container-eqp78m0kgaq0m57]
		2..2...2..2..2..2..
		lens: 4
		v.Type: ContainersReady
		v.Status: False
		v.Reason: ContainersNotReady
		v.Message: containers with unready status: [gpu0-pod-eqp78m0kgaq0m57-container-eqp78m0kgaq0m57]
		3..3...3..3..3..3..
		lens: 4
		v.Type: PodScheduled
		v.Status: True
		v.Reason:
		v.Message:
		0..0...0..0..0..0..
		lens: 4
		v.Type: Initialized
		v.Status: True
		v.Reason:
		v.Message:
		1..1...1..1..1..1..
		lens: 4
		v.Type: Ready
		v.Status: False
		v.Reason: ContainersNotReady
		v.Message: containers with unready status: [gpu0-pod-eqp78m0kgaq0m57-container-eqp78m0kgaq0m57]
		2..2...2..2..2..2..
		lens: 4
		v.Type: ContainersReady
		v.Status: False
		v.Reason: ContainersNotReady
		v.Message: containers with unready status: [gpu0-pod-eqp78m0kgaq0m57-container-eqp78m0kgaq0m57]
		3..3...3..3..3..3..
		lens: 4
		v.Type: PodScheduled
		v.Status: True
		v.Reason:
		v.Message:
		0..0...0..0..0..0..
		lens: 4
		v.Type: Initialized
		v.Status: True
		v.Reason:
		v.Message:
		1..1...1..1..1..1..
		lens: 4
		v.Type: Ready
		v.Status: True
		v.Reason:
		v.Message:
		2..2...2..2..2..2..
		lens: 4
		v.Type: ContainersReady
		v.Status: True
		v.Reason:
		v.Message:
		3..3...3..3..3..3..
		lens: 4
		v.Type: PodScheduled
		v.Status: True
		v.Reason:
		v.Message:
		TRACE: 2021/01/04 10:14:53 service-handler.go:49: created gpu1-svc-eqp78m0kgaq0m57
		TRACE: 2021/01/04 10:14:53 pod-handler.go:120: created gpu1-pod-eqp78m0kgaq0m57
		0..0...0..0..0..0..
		lens: 1
		v.Type: PodScheduled
		v.Status: False
		v.Reason: Unschedulable
		v.Message: 0/9 nodes are available: 2 node(s) didn't match node selector, 7 Insufficient nvidia.com/gpu.
		0..0...0..0..0..0..
		lens: 1
		v.Type: PodScheduled
		v.Status: False
		v.Reason: Unschedulable
		v.Message: 0/9 nodes are available: 2 node(s) didn't match node selector, 7 Insufficient nvidia.com/gpu.
		0..0...0..0..0..0..
		lens: 1
		v.Type: PodScheduled
		v.Status: False
		v.Reason: Unschedulable
		v.Message: 0/9 nodes are available: 2 node(s) didn't match node selector, 7 Insufficient nvidia.com/gpu.
	*/
	//}
	if len(podconditions) == 1 &&
		podconditions[0].Type == apiv1.PodScheduled &&
		podconditions[0].Status == apiv1.ConditionFalse {
		// 资源不充足的pending
		*statusType = INSUFFICIENTPENDING
		Trace.Println(podconditions[0].Reason)
		Trace.Println(podconditions[0].Message)
		Trace.Printf("%s insufficient resouces...\n", podName)
	} else if len(podconditions) == 4 {
		Trace.Printf("%s resouces pass...\n", podName)
	}
	/*flag := 0
	for true {
		if flag == 1 {
			break
		}
		podv1, _ := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
		if podv1.Status.Phase == apiv1.PodPending {
			containerstatues := podv1.Status.ContainerStatuses // 1
			fmt.Printf("tttttttttttttttttt:", len(containerstatues))
			for _, v := range containerstatues {
				//v.State.Terminated.Reason
				//v.State.Terminated.Message
				if v.State.Waiting.Reason == "ContainerCreating" {
					Trace.Printf("ContainerCreating...\n")
					continue
				} else {
					Trace.Printf("ContainerCreating done...\n")
					flag = 1
				}
			}
			podconditions := podv1.Status.Conditions // 4
			fmt.Printf("sssssssssssssss:", len(podconditions))
			for _, v := range podconditions {
				if v.Type == apiv1.PodScheduled { // creating阶段PodScheduled也是true
					if v.Status == apiv1.ConditionFalse {
						fmt.Printf("v.Status: %s\n", v.Status)
						fmt.Printf("v.Reason: %s\n", v.Reason)
						fmt.Printf("v.Message: %s\n", v.Message)
					} else {
						fmt.Printf("xxxxxxxxxxxxxxxxxxxxxxxxxx:PENDING 11\n")
						fmt.Printf("v.Status: %s\n", v.Status)
						fmt.Printf("v.Reason: %s\n", v.Reason)
						fmt.Printf("v.Message: %s\n", v.Message)
					}
				}
			}
		} else if podv1.Status.Phase == apiv1.PodRunning {
			break
		} else if podv1.Status.Phase == apiv1.PodSucceeded {

		} else if podv1.Status.Phase == apiv1.PodFailed {

		} else if podv1.Status.Phase == apiv1.PodUnknown {

		}
		time.Sleep(time.Second * 1)
	}*/
	//podv1, _ := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
	/*
		podCondition[0].Status:False
		podCondition[0].Message:0/3 nodes are available: 3 Insufficient nvidia.com/gpu.
		podCondition[0].Reason:Unschedulable
		podCondition[0].podPhase:Pending
	*/
	//return podv1.Status.Phase
	return podv1.Status.Phase
}

func get_10G_ips(podClient v1.PodInterface, podName string) string {
	podv1, _ := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
	annotations := podv1.GetAnnotations()

	for k, v := range annotations {
		//fmt.Println(k, v)
		if k == "k8s.v1.cni.cncf.io/networks-status" {
			vv := strings.Fields(v)
			for _, ips := range vv {
				if strings.Contains(ips, MATCHIPS) {
					return trimQuotes(ips)
				}
			}
			break
		}
	}
	return ""
	//return podv1.Status.PodIP
}

func getIpFromIppool() string {
	Trace.Println("Assign speed up training ips...")
	for i := 20; i <= 200; i++ {
		assembleIP := MATCHIPS + strconv.Itoa(i)

		if _, ok := IP_POOL[assembleIP]; ok {
			if IP_POOL[assembleIP] == true {
				continue
			} else {
				Trace.Println("Assigned speed up training ips: ", assembleIP)
				IP_POOL[assembleIP] = true
				return assembleIP
			}
		} else {
			Trace.Println("Assigned speed up training ips: ", assembleIP)
			IP_POOL[assembleIP] = true
			return assembleIP
		}
	}
	return ""
}

func writeIppoolToFile() {
	file, error := os.OpenFile(".ippool", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0766)
	if error != nil {
		fmt.Println(error)
	}
	defer file.Close()
	dataReady, err := json.Marshal(IP_POOL)
	if err != nil {
		Trace.Printf("IP_POOL MarshalIndent err: %s\n", err)
	}
	file.Write(dataReady)
}

func writeIppool(ips string) {
	reg := strings.Split(ips, ",")
	for _, ip := range reg {
		if ip == "" {
			break
		}
		IP_POOL[string(ip)] = false
	}
	writeIppoolToFile()
}
