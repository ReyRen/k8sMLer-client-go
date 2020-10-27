package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"strings"
)

func PodReady(pods *apiv1.Pod, podName string, tmpString string, labelName string, gpuQuantity int64, gracePeriodSeconds *int64, pvcName string, currentI int, totalI int) {

	// assemble a container name
	containName := podName + "-container-" + tmpString

	// multus-cni for different interface in pods
	multus := make(map[string]string)
	multus["k8s.v1.cni.cncf.io/networks"] = "macvlan-conf"

	// volumeMount
	mountPath := MOUNTPATH
	mountName := podName + "-mount-" + tmpString

	// get the execute args
	var args []string
	if currentI == totalI-1 {
		// last one pod
		args = []string{MASTER_TAIL}
	} else {
		args = []string{CHILD_TAIL}
	}

	// assemble a resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	var resourceQuantity resource.Quantity
	resourceQuantity.Set(gpuQuantity)
	resourceLimit["nvidia.com/gpu"] = resourceQuantity

	*pods = apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Labels:      map[string]string{labelName: labelName},
			Annotations: multus, // need for multus-cni
		},
		Spec: apiv1.PodSpec{
			Volumes: []apiv1.Volume{
				{
					Name: mountName,
					VolumeSource: apiv1.VolumeSource{
						PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
			Containers: []apiv1.Container{
				{
					Name:       containName,
					Image:      IMAGE,
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
						TerminationMessageFallbackToLogsOnError will read the most recent contents of the container logs
						for the container status message when the container exits with an error and the
						terminationMessagePath has no contents.
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

func Create_pod(podClient v1.PodInterface,
	podName string,
	tmpString string,
	labelName string,
	gpuQuantity int64,
	gracePeriodSeconds *int64,
	pvcName string,
	currentI int,
	totalI int) {
	var pod apiv1.Pod

	PodReady(&pod, podName, tmpString, labelName, gpuQuantity, gracePeriodSeconds, pvcName, currentI, totalI)
	_, err := podClient.Create(context.TODO(), &pod, metav1.CreateOptions{})
	if err != nil {
		Error.Println("create the pod err : ", err)
	}
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

func Get_pod_status(podClient v1.PodInterface, podName string) apiv1.PodPhase {

	podv1, _ := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
	/*
		podCondition[0].Status:False
		podCondition[0].Message:0/3 nodes are available: 3 Insufficient nvidia.com/gpu.
		podCondition[0].Reason:Unschedulable
		podCondition[0].podPhase:Pending
	*/
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
}
