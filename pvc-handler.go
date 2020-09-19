package main

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
