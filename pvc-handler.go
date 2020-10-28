package main

import (
	"context"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func PvcReady(pvcs *apiv1.PersistentVolumeClaim,
	pvcName string,
	tmpString string,
	labelName string,
	cap string) string {

	storageclassName := STORAGECLASS

	// assemble pvc name
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
			StorageClassName: &storageclassName,
			VolumeMode:       nil, // by default is the raw block
		},
	}
	return pvcName
}

func Create_pvc(pvcClient v1.PersistentVolumeClaimInterface,
	pvcName string,
	tmpString string,
	labelName string,
	caps string) string {

	var pvcs apiv1.PersistentVolumeClaim

	realName := PvcReady(&pvcs, pvcName, tmpString, labelName, caps)
	resultPVC, err := pvcClient.Create(context.TODO(), &pvcs, metav1.CreateOptions{})
	if err != nil {
		Error.Println("create the pvc err : ", err)
	}
	Trace.Printf("created %s\n", resultPVC.GetObjectMeta().GetName())
	return realName
}

func List_pvc(pvcClient v1.PersistentVolumeClaimInterface, labelName string) {
	Trace.Println("list persistentvolumeclaim...")
	list, err := pvcClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelName})
	if err != nil {
		Error.Println("list pvc err: ", err)
	}
	for _, s := range list.Items {
		Trace.Printf(" * [%s] pvc in [%s] with [%v] label\n", s.Name, s.Namespace, s.Labels)
	}
}

func Delete_pvc(pvcClient v1.PersistentVolumeClaimInterface, pvcName string, labelName string, gracePeriodSeconds *int64) {

	deletePolicy := metav1.DeletePropagationForeground
	if pvcName != "" {
		if err := pvcClient.Delete(context.TODO(), pvcName, metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}); err != nil {
			Error.Println("delete pvc err:", err)
		}
		Trace.Printf("deleted pvc %s\n", pvcName)
	} else {
		if err := pvcClient.DeleteCollection(context.TODO(), metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriodSeconds,
			PropagationPolicy:  &deletePolicy,
		}, metav1.ListOptions{
			LabelSelector: labelName,
		}); err != nil {
			Error.Println("delete pvcs err:", err)
		}
		Trace.Printf("delete all pvcs under label: %s\n", labelName)
	}
}
