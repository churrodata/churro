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
	"os"

	"github.com/churrodata/churro/api/v1alpha1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	yaml "sigs.k8s.io/yaml"
)

const (
	podName = "churro-extractsource"
)

func (r PipelineReconciler) processExtractSource(req ctrl.Request, pipeline v1alpha1.Pipeline) error {
	// get referenced churro-extractsource Pod objects
	var pod v1.Pod
	var podFound bool
	thing := types.NamespacedName{
		Namespace: pipeline.ObjectMeta.Namespace,
		Name:      podName,
	}
	err := r.Get(r.Ctx, thing, &pod)
	if err != nil {
		r.Log.Error(err, "unable to get churro-extractsource Pod")
	} else {
		podFound = true
	}

	// create the churro-extractsource Pod if necessary
	if !podFound {
		var pod v1.Pod
		err := yaml.Unmarshal(r.ExtractSourcePodTemplate, &pod)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal churro-extractsource pod template")
			return err
		}

		pullSecretName := os.Getenv("CHURRO_PULL_SECRET_NAME")
		if pullSecretName != "" {
			ref := v1.LocalObjectReference{
				Name: pullSecretName,
			}
			pod.Spec.ImagePullSecrets = []v1.LocalObjectReference{ref}
		}

		pod.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name, "service": podName}
		pod.Namespace = pipeline.ObjectMeta.Namespace
		pod.Name = podName
		pEnv := v1.EnvVar{Name: "CHURRO_PIPELINE", Value: pipeline.ObjectMeta.Name}
		if pullSecretName != "" {
			pEnv2 := v1.EnvVar{Name: "CHURRO_PULL_SECRET_NAME", Value: pullSecretName}
			pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, pEnv2)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, pEnv)

		if err := ctrl.SetControllerReference(&pipeline, &pod, r.Scheme); err != nil {
			r.Log.Error(err, "error setting controller reference")
			return err
		}
		if err := r.Create(r.Ctx, &pod); err != nil {
			r.Log.Error(err, "unable to create churro-extractsource pod for pipeline", "pod", pod)
			return err
		}
		r.Log.V(1).Info("created churro-extractsource pod for pipeline ")
	}

	return nil
}
