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
	"fmt"
	"os"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/pkg"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	cockroachClientSecret = "cockroachdb.client.root"
	cockroachNodeSecret   = "cockroachdb.node"
	churroGRPCSecret      = "churro.client.root"
	churroConfigSecret    = "churro.config"
)

func (r PipelineReconciler) processSecrets(pipeline v1alpha1.Pipeline) error {

	// get the most recent pipeline

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	pipelineClient, err := pkg.NewClient(config, pipeline.ObjectMeta.Namespace)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	pipelineToUpdate, err := pipelineClient.Get(pipeline.ObjectMeta.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	// get referenced secrets objects
	var childSecrets v1.SecretList
	err = r.List(r.Ctx, &childSecrets, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child Secret")
		return err
	}

	// compare referenced Secret objects with what we expect
	// make sure we have a secret/cockroachdb.client.root
	needCockroachClientSecret := true
	// make sure we have a secret/cockroachdb.node
	needCockroachNodeSecret := true
	// make sure we have a secret/churro.client.root
	needChurroGRPCSecret := true

	// make sure we have a secret/xyz for the image pull secret if set
	needChurroImagePullSecret := true

	pullSecretName := os.Getenv("CHURRO_PULL_SECRET_NAME")
	for i := 0; i < len(childSecrets.Items); i++ {
		r := childSecrets.Items[i]
		switch r.Name {
		case cockroachClientSecret:
			needCockroachClientSecret = false
		case cockroachNodeSecret:
			needCockroachNodeSecret = false
		case churroGRPCSecret:
			needChurroGRPCSecret = false
		case pullSecretName:
			needChurroImagePullSecret = false
		}
	}

	// create any expected secrets , set owner reference to this pipeline
	secretsToCreate := make([]v1.Secret, 0)
	if needCockroachClientSecret {
		secret := v1.Secret{}
		secret.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		secret.Name = cockroachClientSecret
		err := doDBCreds(&secret, pipeline.Name, pipelineToUpdate.Spec.DatabaseCredentials)
		if err != nil {
		}
		secret.Namespace = pipeline.ObjectMeta.Namespace
		secretsToCreate = append(secretsToCreate, secret)
	}
	if needCockroachNodeSecret {
		secret := v1.Secret{}
		secret.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		secret.Name = cockroachNodeSecret
		err := doDBNodeCreds(&secret, pipelineToUpdate.Spec.DatabaseCredentials)
		if err != nil {
			return err
		}
		secret.Namespace = pipeline.ObjectMeta.Namespace
		secretsToCreate = append(secretsToCreate, secret)
	}
	if needChurroGRPCSecret {
		secret := v1.Secret{}
		secret.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		secret.Name = churroGRPCSecret
		secret.Namespace = pipeline.ObjectMeta.Namespace
		err := doServiceCreds(&secret, pipelineToUpdate.Spec.ServiceCredentials)
		if err != nil {
			return err
		}

		secretsToCreate = append(secretsToCreate, secret)
	}

	if needChurroImagePullSecret {

		// get the pullImageSecret from the churro namespace
		churrosecret := v1.Secret{}
		thing := types.NamespacedName{
			Namespace: os.Getenv("CHURRO_NAMESPACE"),
			Name:      pullSecretName,
		}

		err := r.Get(r.Ctx, thing, &churrosecret)
		if err != nil {
			return err
		}

		churrosecret.Namespace = pipeline.ObjectMeta.Namespace
		churrosecret.ObjectMeta.ResourceVersion = ""
		secretsToCreate = append(secretsToCreate, churrosecret)

		// create a copy of that secret in the pipeline namespace
	}

	for _, secret := range secretsToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &secret, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &secret); err != nil {
			r.Log.Error(err, "unable to create Secret for pipeline", "service", secret)
			return err
		}
		log.Info().Msg("created Secret for pipeline ")
	}

	return nil
}

func doDBNodeCreds(secret *v1.Secret, dbCreds v1alpha1.DBCreds) error {
	// key should be file name in dir
	// data should be base64 encoded should be file contents
	secret.Data = make(map[string][]byte, 0)

	// node.key
	if len(dbCreds.NodeKey) == 0 {
		return fmt.Errorf("node.key is empty")
	}
	secret.Data["node.key"] = []byte(dbCreds.NodeKey)
	log.Info().Msg(fmt.Sprintf("node.key bytes %d\n", len(dbCreds.NodeKey)))

	if len(dbCreds.NodeCrt) == 0 {
		return fmt.Errorf("node.crt is empty")
	}

	secret.Data["node.crt"] = []byte(dbCreds.NodeCrt)
	log.Info().Msg(fmt.Sprintf("node.crt bytes %d\n", len(dbCreds.NodeCrt)))

	// ca.crt
	if len(dbCreds.CACrt) == 0 {
		return fmt.Errorf("ca.crt is empty")
	}
	secret.Data["ca.crt"] = []byte(dbCreds.CACrt)
	log.Info().Msg(fmt.Sprintf("ca.crt bytes %d\n", len(dbCreds.CACrt)))
	// ca.key
	if len(dbCreds.CAKey) == 0 {
		return fmt.Errorf("ca.key is empty")
	}
	secret.Data["ca.key"] = []byte(dbCreds.CAKey)
	log.Info().Msg(fmt.Sprintf("ca.key bytes %d\n", len(dbCreds.CAKey)))

	secret.Data["client.root.key"] = []byte(dbCreds.ClientRootKey)
	log.Info().Msg(fmt.Sprintf("client.root.key bytes %d\n", len(dbCreds.ClientRootKey)))
	// client.root.crt
	if len(dbCreds.ClientRootCrt) == 0 {
		return fmt.Errorf("client.root.crt is empty")
	}
	secret.Data["client.root.crt"] = []byte(dbCreds.ClientRootCrt)
	log.Info().Msg(fmt.Sprintf("client.root.crt bytes %d\n", len(dbCreds.ClientRootCrt)))
	// client.somepipeline.crt
	if len(dbCreds.ClientRootCrt) == 0 {
		return fmt.Errorf("client.pipeline.crt is empty")
	}
	return nil
}

func doDBCreds(secret *v1.Secret, pipelineName string, dbCreds v1alpha1.DBCreds) error {
	// key should be file name in dir
	// data should be base64 encoded should be file contents
	secret.Data = make(map[string][]byte, 0)

	// ca.crt
	if len(dbCreds.CACrt) == 0 {
		return fmt.Errorf("ca.crt is empty")
	}
	secret.Data["ca.crt"] = []byte(dbCreds.CACrt)
	log.Info().Msg(fmt.Sprintf("ca.crt bytes %d\n", len(dbCreds.CACrt)))
	// ca.key
	if len(dbCreds.CAKey) == 0 {
		return fmt.Errorf("ca.key is empty")
	}
	secret.Data["ca.key"] = []byte(dbCreds.CAKey)
	log.Info().Msg(fmt.Sprintf("ca.key bytes %d\n", len(dbCreds.CAKey)))
	// client.root.key
	if len(dbCreds.ClientRootKey) == 0 {
		return fmt.Errorf("client.root.key is empty")
	}
	secret.Data["client.root.key"] = []byte(dbCreds.ClientRootKey)
	log.Info().Msg(fmt.Sprintf("client.root.key bytes %d\n", len(dbCreds.ClientRootKey)))
	// client.root.crt
	if len(dbCreds.ClientRootCrt) == 0 {
		return fmt.Errorf("client.root.crt is empty")
	}
	secret.Data["client.root.crt"] = []byte(dbCreds.ClientRootCrt)
	log.Info().Msg(fmt.Sprintf("client.root.crt bytes %d\n", len(dbCreds.ClientRootCrt)))
	// client.somepipeline.crt
	if len(dbCreds.ClientRootCrt) == 0 {
		return fmt.Errorf("client.pipeline.crt is empty")
	}
	secret.Data["client."+pipelineName+".crt"] = []byte(dbCreds.PipelineCrt)
	log.Info().Msg(fmt.Sprintf("client."+pipelineName+".crt bytes %d\n", len(dbCreds.PipelineCrt)))
	// client.somepipeline.key
	if len(dbCreds.ClientRootKey) == 0 {
		return fmt.Errorf("client.pipeline.key is empty")
	}
	secret.Data["client."+pipelineName+".key"] = []byte(dbCreds.PipelineKey)
	log.Info().Msg(fmt.Sprintf("client."+pipelineName+".key bytes %d\n", len(dbCreds.PipelineKey)))
	return nil
}

func doServiceCreds(secret *v1.Secret, serviceCreds v1alpha1.ServiceCreds) error {
	// key should be file name in dir
	// data should be base64 encoded should be file contents
	secret.Data = make(map[string][]byte, 0)

	if len(serviceCreds.ServiceCrt) == 0 {
		return fmt.Errorf("service.crt is empty")
	}
	if len(serviceCreds.ServiceKey) == 0 {
		return fmt.Errorf("service.key is empty")
	}

	secret.Data["service.crt"] = []byte(serviceCreds.ServiceCrt)
	log.Info().Msg(fmt.Sprintf("service.crt bytes %d\n", len(serviceCreds.ServiceCrt)))

	secret.Data["service.key"] = []byte(serviceCreds.ServiceKey)
	log.Info().Msg(fmt.Sprintf("service.key bytes %d\n", len(serviceCreds.ServiceKey)))

	return nil
}
