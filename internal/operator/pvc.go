// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package operator

import (
	"context"
	"errors"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/rs/zerolog/log"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
)

const (
	churroDataPVC = "churrodata"
)

func (r PipelineReconciler) processPVCs(pipeline v1alpha1.Pipeline) error {
	// get referenced PVC objects
	var childPVCs v1.PersistentVolumeClaimList
	err := r.List(r.Ctx, &childPVCs, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child PVCs")
		return err
	}

	// compare referenced PVC objects with what we expect
	// make sure we have a pvc/churrodata
	needChurrodataPVC := true

	for i := 0; i < len(childPVCs.Items); i++ {
		r := childPVCs.Items[i]
		switch r.Name {
		case churroDataPVC:
			needChurrodataPVC = false
		}
	}

	// create any expected PVC objects, set owner reference to this pipeline
	pvcsToCreate := make([]v1.PersistentVolumeClaim, 0)
	if needChurrodataPVC {
		var pvc v1.PersistentVolumeClaim
		err := yaml.Unmarshal(r.PVCTemplate, &pvc)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal PVC template")
			return err
		}

		pvc.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		pvc.Name = churroDataPVC
		pvc.Namespace = pipeline.ObjectMeta.Namespace

		if pipeline.Spec.StorageClassName == "" {
			v := getDefaultSC(r.Log)
			pvc.Spec.StorageClassName = &v
		}

		if pipeline.Spec.StorageClassName != "" {
			pvc.Spec.StorageClassName = &pipeline.Spec.StorageClassName
		}
		if pipeline.Spec.StorageSize != "" {
			qty, err := resource.ParseQuantity(pipeline.Spec.StorageSize)
			if err != nil {
				r.Log.Error(err, "unable to parse storage size")
				return err
			}
			pvc.Spec.Resources.Requests[v1.ResourceStorage] = qty
		} else {
			qty, err := resource.ParseQuantity("1G")
			if err != nil {
				r.Log.Error(err, "unable to parse storage size for default size")
				return err
			}

			pvc.Spec.Resources.Requests[v1.ResourceStorage] = qty
		}

		switch pipeline.Spec.AccessMode {
		case "", "ReadWriteMany":
			pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteMany}
		case "ReadWriteOnce":
			pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
		default:
			r.Log.Error(errors.New("accessmode is not right"+pipeline.Spec.AccessMode), "error in AccessMode")
			return err
		}

		/**
		apiVersion: v1
		kind: PersistentVolumeClaim
		metadata:
		  annotations:
		    volume.beta.kubernetes.io/storage-provisioner: rancher.io/local-path
		    volume.kubernetes.io/selected-node: roaster
		  labels:
		    app: churro
		    pipeline: pipeline1
		  name: churrodata
		spec:
		  accessModes:
		  - ReadWriteOnce
		  resources:
		    requests:
		      storage: 100M
		  storageClassName: local-path
		*/
		pvcsToCreate = append(pvcsToCreate, pvc)
	}

	for _, pvc := range pvcsToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &pvc, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &pvc); err != nil {
			r.Log.Error(err, "unable to create PVC for pipeline", "pvc", pvc)
			return err
		}
		log.Info().Msg("created pvc for pipeline ")
	}

	return nil
}

func getDefaultSC(mylog logr.Logger) string {
	clientset, _, err := GetKubeClient()
	if err != nil {
		mylog.Error(err, "unable to connect to kube")
		return ""
	}
	var storageClasses *storagev1.StorageClassList
	storageClasses, err = clientset.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		mylog.Error(err, "unable to list storageclasses")
		return ""
	}

	for _, sc := range storageClasses.Items {
		log.Info().Msg("storageClasses " + sc.Name)
		if sc.ObjectMeta.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			mylog.Info("found default storage class of " + sc.Name)
			return sc.Name
		}
	}

	mylog.Info("could not find a default storage class on this cluster")

	return ""

}
