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
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r PipelineReconciler) processPDBs(pipeline v1alpha1.Pipeline) error {
	// get referenced pdb objects
	var childPDBs v1beta1.PodDisruptionBudgetList
	err := r.List(r.Ctx, &childPDBs, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child pdb")
		return err
	}

	// compare referenced pdb objects with what we expect
	// make sure we have a pdb/cockroachdb-budget
	needCockroachPDB := true
	for i := 0; i < len(childPDBs.Items); i++ {
		r := childPDBs.Items[i]
		if r.Name == "cockroachdb-budget" {
			needCockroachPDB = false
		}
	}

	// create any expected pdb objects, set owner reference to this pipeline
	pdbsToCreate := make([]v1beta1.PodDisruptionBudget, 0)
	if needCockroachPDB {
		pdb := v1beta1.PodDisruptionBudget{}
		pdb.Name = "cockroachdb-budget"
		pdb.Namespace = pipeline.ObjectMeta.Namespace
		/**
		apiVersion: policy/v1beta1
		kind: PodDisruptionBudget
		metadata:
		  name: cockroachdb-budget
		  labels:
		    app: cockroachdb
		spec:
		  selector:
		    matchLabels:
		      app: cockroachdb
		  maxUnavailable: 1
		*/

		pdb.ObjectMeta.Name = pdb.Name
		pdb.ObjectMeta.Labels = map[string]string{"app": "cockroachdb"}

		ls := metav1.LabelSelector{}
		ls.MatchLabels = map[string]string{"app": "cockroachdb"}
		pdb.Spec.Selector = &ls
		v := intstr.FromInt(1)
		pdb.Spec.MaxUnavailable = &v

		pdbsToCreate = append(pdbsToCreate, pdb)
	}
	for _, pdb := range pdbsToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &pdb, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &pdb); err != nil {
			r.Log.Error(err, "unable to create pdb for pipeline", "pdb", pdb)
			return err
		}
		r.Log.Info("created pdb for pipeline ")
	}

	return nil
}
