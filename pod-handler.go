package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
	"time"
)

func PodReady(pods *apiv1.Pod, podName string, tmpString string, labelName string, gpuQuantity int64, gracePeriodSeconds *int64, pvcName string) {
	// assemble a container name
	containName := podName + "-container-" + tmpString
	// assemble a pod name
	podName = podName + "-pod-" + tmpString

	// volumeMount
	mountPath := "/usr/share/horovod"
	mountName := podName + "-mount-" + tmpString

	// assemble a resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	var resourceQuantity resource.Quantity
	resourceQuantity.Set(gpuQuantity)
	resourceLimit["nvidia.com/gpu"] = resourceQuantity

	*pods = apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Labels:      map[string]string{labelName: labelName},
			Annotations: nil, // need for multus-cni
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
					Name:    containName,
					Image:   "horovod/horovod:0.18.1-tf1.14.0-torch1.2.0-mxnet1.5.0-py3.6", // testing
					Command: []string{"/bin/sh", "-c"},
					//Args:    []string{"python tensorflow_mnist.py", "tail -f /dev/null"},
					Args:       []string{"tail -f /dev/null"},
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

func Create_pod(podClient v1.PodInterface, podName string, tmpString string, labelName string, gpuQuantity int64, gracePeriodSeconds *int64, pvcName string) {
	var pod apiv1.Pod

	PodReady(&pod, podName, tmpString, labelName, gpuQuantity, gracePeriodSeconds, pvcName)
	fmt.Println("creating pod...")
	result, err := podClient.Create(context.TODO(), &pod, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("create the pod err : ", err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
}

func List_pod(podClient v1.PodInterface, labelName string) {
	fmt.Println("list pod...")
	list, err := podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
	if err != nil {
		log.Fatalln("list pod err: ", err)
	}
	for _, s := range list.Items {
		fmt.Printf(" * [%s] pod in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
	}
}

func Delete_pod(podClient v1.PodInterface, podName string, labelName string, gracePeriodSeconds *int64) {
	deletePolicy := metav1.DeletePropagationForeground
	if podName != "" {
		fmt.Println("delete pod...")
		if err := podClient.Delete(context.TODO(), podName, metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}); err != nil {
			log.Fatalln("delete pod err:", err)
		}
		fmt.Printf("deleted pod %s\n", podName)
	} else {
		fmt.Println("delete pods...")
		if err := podClient.DeleteCollection(context.TODO(), metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
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
}

func Get_pod_status(podClient v1.PodInterface, podName string) (apiv1.ConditionStatus, string, string, apiv1.PodPhase) {
	var podv1 *apiv1.Pod

	podv1, _ = podClient.Get(context.TODO(), podName, metav1.GetOptions{})
	//a.GetAnnotations()
	podCondition := podv1.Status.Conditions
	for len(podCondition) == 0 {
		time.Sleep(time.Second * 3)
	}
	/*
		podCondition[0].Status:False
		podCondition[0].Message:0/3 nodes are available: 3 Insufficient nvidia.com/gpu.
		podCondition[0].Reason:Unschedulable
		podCondition[0].podPhase:Pending
	*/
	return podCondition[0].Status, podCondition[0].Message, podCondition[0].Reason, podv1.Status.Phase

}
