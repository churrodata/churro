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
	"github.com/churrodata/churro/api/v1alpha1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	yaml "sigs.k8s.io/yaml"
)

func (r PipelineReconciler) processCockroachClient(req ctrl.Request, pipeline v1alpha1.Pipeline) error {
	const podName = "cockroachdb-client-secure"
	// get referenced Pod object
	var pod v1.Pod
	var podFound bool
	thing := types.NamespacedName{
		Namespace: pipeline.ObjectMeta.Namespace,
		Name:      podName,
	}
	err := r.Get(r.Ctx, thing, &pod)
	if err != nil {
		r.Log.Error(err, "unable to get "+podName)
	} else {
		podFound = true
	}

	// create the Pod if necessary
	if !podFound {
		var pod v1.Pod
		err := yaml.Unmarshal(r.CockroachClientPodTemplate, &pod)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal "+podName)
			return err
		}

		pod.Namespace = pipeline.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&pipeline, &pod, r.Scheme); err != nil {
			r.Log.Error(err, "error setting controller reference")
			return err
		}
		if err := r.Create(r.Ctx, &pod); err != nil {
			r.Log.Error(err, "unable to create "+podName+" pod for pipeline", "pod", pod)
			return err
		}
		r.Log.V(1).Info("created " + podName + " pod for pipeline ")
	}

	return nil
}
