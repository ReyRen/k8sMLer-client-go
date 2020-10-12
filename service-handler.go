package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
)

func ServiceReady(service *apiv1.Service, serviceName string, labelName string, gracePeriodSeconds *int64) string {
	// assemble a pod name
	//serviceName = serviceName + "-svc-" + tmpString

	*service = apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			Labels: map[string]string{
				labelName: labelName,
			},
			Annotations:                nil,
			DeletionGracePeriodSeconds: gracePeriodSeconds,
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

	return serviceName
}

func Create_service(svcClient v1.ServiceInterface, serviceName string, labelName string, gracePeriodSeconds *int64) string {
	var service apiv1.Service

	realSvcName := ServiceReady(&service, serviceName, labelName, gracePeriodSeconds)
	fmt.Println("creating service...")
	result, err := svcClient.Create(context.TODO(), &service, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("create the service err : ", err)
	}
	fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
	return realSvcName
}

func List_service(svcClient v1.ServiceInterface, labelName string) {
	fmt.Println("list service...")
	list, err := svcClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
	if err != nil {
		log.Fatalln("list svc err: ", err)
	}
	for _, s := range list.Items {
		fmt.Printf(" * [%s] svc in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
	}
}

func Delete_service(svcClient v1.ServiceInterface, serviceName string, gracePeriodSeconds *int64) {
	fmt.Println("delete service...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := svcClient.Delete(context.TODO(), serviceName, metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSeconds,
		PropagationPolicy:  &deletePolicy,
	}); err != nil {
		log.Fatalln("delete svc err: ", err)
	}
	fmt.Printf("deleted service %s\n", serviceName)
}
