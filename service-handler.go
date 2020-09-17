package main

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ServiceReady(service *apiv1.Service, serviceName string, labelName string) {
	*service = apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			Labels: map[string]string{
				labelName: labelName,
			},
			Annotations: nil,
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name:        "ssh",
					Protocol:    apiv1.ProtocolTCP,
					AppProtocol: nil,
					Port:        22,
					TargetPort:  intstr.IntOrString{IntVal: 22},
				},
			},
			Selector: map[string]string{
				"app": labelName,
			},
		},
		//Status:     apiv1.ServiceStatus{},
	}
}
