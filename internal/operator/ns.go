// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package operator holds the churro operator logic
package operator

import (
	"fmt"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProcessCR handles a reconcile event
func (r PipelineReconciler) ProcessCR(req ctrl.Request) (result ctrl.Result, err error) {

	// get the CR
	var pipeline v1alpha1.Pipeline
	if err := r.Get(r.Ctx, req.NamespacedName, &pipeline); err != nil {
		r.Log.Error(err, "unable to fetch pipeline")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.

		// clean up the namespace we created for the pipeline
		ns := v1.Namespace{}
		ns.Name = req.Namespace

		if err := r.Delete(r.Ctx, &ns); err != nil {
			r.Log.Error(err, "unable to delete Namespace for pipeline", "namespace", ns)
			return result, err
		}

		return result, client.IgnoreNotFound(err)
	}
	r.Log.Info("got a pipeline " + pipeline.Name)
	err = r.processPipeline(req, pipeline)
	if err != nil {
		r.Log.Error(err, "error in processing pipeline")
	}
	return result, err
}

func (r PipelineReconciler) processPipeline(req ctrl.Request, pipeline v1alpha1.Pipeline) error {
	// insure pipeline ID is generated
	err := r.processId(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processId")
	}
	// insure creds are generated for pipeline
	err = r.processCreds(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processCreds")
	}
	// insure rbac is defined for the pipeline
	err = r.processRoles(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processRoles")
	}
	err = r.processRoleBindings(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processRoleBindings")
	}

	err = r.processServiceAccounts(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processServiceAccounts")
	}
	err = r.processServices(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processServices")
	}
	err = r.processPVCs(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processPVCs")
	}
	err = r.processSecrets(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processSecrets")
	}
	err = r.processPDBs(pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processPDBs")
	}
	switch pipeline.Spec.DatabaseType {
	case domain.DatabaseCockroach:
		err = r.processStatefulSet(pipeline)
		if err != nil {
			return err
		}
		err = r.initStatefulSet(req, pipeline)
		if err != nil {
			return err
		}
		err = r.processCockroachClient(req, pipeline)
		if err != nil {
			return err
		}
	case domain.DatabaseMysql:
		//	case db.CHURRODB_MYSQL:
		err = r.processMysql(pipeline)
		if err != nil {
			//return err
			r.Log.Error(err, "error in Mysql")
		}
	case domain.DatabaseSinglestore:
		//	case db.CHURRODB_SINGLESTORE:
		err = r.processSinglestore(pipeline)
		if err != nil {
			//return err
			r.Log.Error(err, "error in Singlestore")
		}
		err = r.processSinglestoreClient(req, pipeline)
		if err != nil {
			//return err
			r.Log.Error(err, "error in Singlestore client")
		}
	default:
		return fmt.Errorf("unsupported databasetype %s", pipeline.Spec.DatabaseType)
	}
	err = r.processExtractSource(req, pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processExtractSource")
	}
	err = r.processCtl(req, pipeline)
	if err != nil {
		//return err
		r.Log.Error(err, "error in processCtl")
	}
	return err
}
