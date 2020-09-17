package main

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PodReady(pods *apiv1.Pod, podName string, labelName string, gpuQuantity int64) {
	// assemble a container name
	tmpString := GetRandomString(15)
	containName := podName + "-container-" + tmpString

	// assemble a resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	var resourceQuantity resource.Quantity
	resourceQuantity.Set(gpuQuantity)
	resourceLimit["nvidia.com/gpu"] = resourceQuantity

	// Termination
	gracePeriodSeconds := int64(0)

	*pods = apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Labels:      map[string]string{labelName: labelName},
			Annotations: nil, // need for multus-cni
		},
		Spec: apiv1.PodSpec{
			Volumes: nil,
			Containers: []apiv1.Container{
				{
					Name:       containName,
					Image:      "horovod/horovod:0.18.1-tf1.14.0-torch1.2.0-mxnet1.5.0-py3.6", // testing
					Command:    []string{"/bin/sh", "-c"},
					Args:       []string{"tail -f /dev/null"},
					WorkingDir: "",
					Ports:      []apiv1.ContainerPort{},
					Env:        nil,
					Resources: apiv1.ResourceRequirements{
						Limits:   resourceLimit,
						Requests: nil,
					},
					VolumeMounts:             nil,
					VolumeDevices:            nil,
					Lifecycle:                nil,
					TerminationMessagePath:   "",
					TerminationMessagePolicy: apiv1.TerminationMessageFallbackToLogsOnError,
					/*
						TerminationMessageFallbackToLogsOnError will read the most recent contents of the container logs
						for the container status message when the container exits with an error and the
						terminationMessagePath has no contents.
					*/
				},
			},
			RestartPolicy:                 "",
			TerminationGracePeriodSeconds: &gracePeriodSeconds,
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
