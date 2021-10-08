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

	b64 "encoding/base64"

	mysqlv1alpha1 "github.com/presslabs/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/rs/zerolog/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	yaml "sigs.k8s.io/yaml"
)

func (r PipelineReconciler) processMysql(pipeline v1alpha1.Pipeline) error {
	// get referenced StatefulSet objects
	var children mysqlv1alpha1.MysqlClusterList
	err := r.List(r.Ctx, &children, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child mysql CRs")
		return err
	}

	// compare referenced StatefulSet objects with what we expect
	// make sure we have a statefulset/cockroachdb
	needCR := true

	for i := 0; i < len(children.Items); i++ {
		r := children.Items[i]
		log.Info().Msg("mysql children name is " + r.Name)
		//switch r.Name {
		//case "cockroachdb":
		needCR = false
		//}
	}

	// create any expected StatefulSet objects, set owner reference to this pipeline
	if needCR {
		// create the mysql Secret
		var mysqlsecret v1.Secret
		err := yaml.Unmarshal(r.MYSQLSecretTemplate, &mysqlsecret)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal mysql secret template")
			return err
		}

		// set password using cr password value

		mysqlsecret.Data["ROOT_PASSWORD"] = []byte(pipeline.Spec.AdminDataSource.Password)

		//mysqlsecret.Data["ROOT_PASSWORD"] = []byte("bm90LXNvLXNlY3VyZQ==")

		mysqlsecret.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		mysqlsecret.Namespace = pipeline.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&pipeline, &mysqlsecret, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &mysqlsecret); err != nil {
			r.Log.Error(err, "unable to create mysql secret for pipeline", "mysqlsecret", mysqlsecret)
			return err
		}
		r.Log.V(1).Info("created mysql secret for pipeline ")

		// create the mysql Custom Resource
		var mysqlcr mysqlv1alpha1.MysqlCluster
		err = yaml.Unmarshal(r.MYSQLCRTemplate, &mysqlcr)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal mysql CR template")
			return err
		}

		mysqlcr.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		mysqlcr.Namespace = pipeline.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&pipeline, &mysqlcr, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &mysqlcr); err != nil {
			r.Log.Error(err, "unable to create mysql cr for pipeline", "mysqlcr", mysqlcr)
			return err
		}
		r.Log.V(1).Info("created mysql cr for pipeline ")
	}

	return nil
}
