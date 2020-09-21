package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
)

func PvcReady(pvcs *apiv1.PersistentVolumeClaim, pvcName string, labelName string, gracePeriodSeconds *int64, cap string) {

	// sc name
	scName := "nfs-sc"

	// annotation key
	//annKey := "volume.beta.kubernetes.io/storage-class"

	// assemble pvc name
	tmpString := GetRandomString(15)
	pvcName = pvcName + "-pvc-" + tmpString

	// assemble resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	totalClaimedQuant := resource.MustParse(cap)
	resourceLimit[apiv1.ResourceStorage] = totalClaimedQuant

	*pvcs = apiv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvcName,
			Labels:      map[string]string{labelName: labelName},
			Annotations: nil,
		},
		Spec: apiv1.PersistentVolumeClaimSpec{
			AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteMany},
			/*Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{labelName: labelName}, // used to binding with pv, if pv create manual
			},*/
			Resources: apiv1.ResourceRequirements{
				Requests: resourceLimit,
			},
			VolumeName:       "", // binding to which PV
			StorageClassName: &scName,
			VolumeMode:       nil, // by default is the raw block
		},
	}
}

func Create_pvc(pvcClient v1.PersistentVolumeClaimInterface, pvcName string, labelName string, gracePeriodSeconds *int64, caps string) {
	var pvcs apiv1.PersistentVolumeClaim

	PvcReady(&pvcs, pvcName, labelName, gracePeriodSeconds, caps)
	fmt.Println("creating pvc...")
	resultPVC, err := pvcClient.Create(context.TODO(), &pvcs, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("create the pvc err : ", err)
	}
	fmt.Printf("Created persistentvolumeclaim %q.\n", resultPVC.GetObjectMeta().GetName())
}

func List_pvc(pvcClient v1.PersistentVolumeClaimInterface, labelName string) {
	fmt.Println("list persistentvolumeclaim...")
	list, err := pvcClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
	if err != nil {
		log.Fatalln("list pvc err: ", err)
	}
	for _, s := range list.Items {
		fmt.Printf(" * [%s] pvc in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
	}
}

func Delete_pvc(pvcClient v1.PersistentVolumeClaimInterface, pvcName string, labelName string, gracePeriodSeconds *int64) {

	deletePolicy := metav1.DeletePropagationForeground
	if pvcName != "" {
		fmt.Println("delete pvc...")
		if err := pvcClient.Delete(context.TODO(), pvcName, metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}); err != nil {
			log.Fatalln("delete pvc err:", err)
		}
		fmt.Printf("deleted pvc %s\n", pvcName)
	} else {
		fmt.Println("delete pvcs...")
		if err := pvcClient.DeleteCollection(context.TODO(), metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}, metav1.ListOptions{
			LabelSelector: labelName,
		}); err != nil {
			log.Fatalln("delete pvcs err:", err)
		}
		fmt.Printf("delete all pvcs under label: %s\n", labelName)
	}
}
