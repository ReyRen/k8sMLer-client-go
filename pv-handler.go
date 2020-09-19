package main

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PvReady(pv *apiv1.PersistentVolume, pvcName string, labelName string, gracePeriodSeconds *int64, cap string) {

	//accemble pv name
	tmpString := GetRandomString(15)
	pvName := pvcName + "-pv-" + tmpString

	// assemble resource limit
	resourceLimit := make(map[apiv1.ResourceName]resource.Quantity)
	totalClaimedQuant := resource.MustParse(cap)
	resourceLimit[apiv1.ResourceStorage] = totalClaimedQuant

	*pv = apiv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       pvName,
			DeletionGracePeriodSeconds: gracePeriodSeconds,
			Labels:                     map[string]string{labelName: labelName},
		},
		Spec: apiv1.PersistentVolumeSpec{
			Capacity: resourceLimit,
			PersistentVolumeSource: apiv1.PersistentVolumeSource{
				NFS: &apiv1.NFSVolumeSource{
					Server:   "192.168.0.113",
					Path:     "/root/data/volumes/v2",
					ReadOnly: false,
				},
			},
			AccessModes:                   []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteMany},
			PersistentVolumeReclaimPolicy: apiv1.PersistentVolumeReclaimRecycle,
		},
	}
}
