// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package pipeline

import (
	"context"
	"fmt"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog/log"
)

// GetPipeline ...
func GetPipeline(namespace string) (v1alpha1.Pipeline, error) {

	var p *v1alpha1.Pipeline

	// connect to the Kube API
	_, config, err := pkg.GetKubeClient()
	if err != nil {
		return *p, err
	}

	pipelineClient, err := pkg.NewClient(config, namespace)
	if err != nil {
		return *p, err
	}

	p, err = pipelineClient.Get(namespace)
	if err != nil {
		return *p, err
	}

	return *p, nil
}

// DeletePipeline ...
func DeletePipeline(pipelineName string) error {
	// connect to the Kube API
	_, config, err := pkg.GetKubeClient()
	if err != nil {
		return err
	}

	pipelineClient, err := pkg.NewClient(config, pipelineName)
	if err != nil {
		return err
	}

	err = pipelineClient.Delete(pipelineName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil

}

// CreatePipeline ...
func CreatePipeline(cr v1alpha1.Pipeline) error {

	pipelineName := cr.ObjectMeta.Name

	// connect to the Kube API
	client, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	// create the pipeline namespace if necessary

	ns, err := client.CoreV1().Namespaces().Get(context.TODO(), pipelineName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		log.Info().Msg("namespace is not found, will create..." + pipelineName)
		// create the namespace
		ns.ObjectMeta.Name = pipelineName
		_, err := client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		if err != nil {
			log.Error().Stack().Err(err).Msg("error creating namespace " + ns.Name)
		}
		log.Info().Msg("created namespace " + pipelineName)

	} else if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	// generate the pipeline credentials

	/**
	rsaBits := 4096
	dur, err := time.ParseDuration("8760h")
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	serviceHosts := fmt.Sprintf("*.%s.svc.cluster.local,churro-watch.%s.svc.cluster.local,churro-ctl.%s.svc.cluster.local,localhost,churro-watch,churro-ctl,127.0.0.1", pipelineName, pipelineName, pipelineName)
	log.Info().Msg("service hosts for credentials are " + serviceHosts)

	dbCreds, err := churroctl.GenerateChurroCreds(pipelineName, serviceHosts, rsaBits, dur)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	*/

	fillCRDefaults(&cr, pipelineName)

	// add the credentials to the CR

	/**
	d := v1alpha1.DBCreds{}
	d.CAKey = string(dbCreds.Cakey)
	d.CACrt = string(dbCreds.Cacrt)
	d.NodeKey = string(dbCreds.Nodekey)
	d.NodeCrt = string(dbCreds.Nodecrt)
	d.ClientRootCrt = string(dbCreds.Clientrootcrt)
	d.ClientRootKey = string(dbCreds.Clientrootkey)
	d.PipelineCrt = string(dbCreds.Clientcrt)
	d.PipelineKey = string(dbCreds.Clientkey)

	cr.Spec.DatabaseCredentials = d

	s := v1alpha1.ServiceCreds{}
	s.ServiceCrt = string(dbCreds.Servicecrt)
	s.ServiceKey = string(dbCreds.Servicekey)

	cr.Spec.ServiceCredentials = s
	*/

	// create the pipeline CR in k8s
	pipelineClient, err := pkg.NewClient(config, pipelineName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	log.Info().Msg(fmt.Sprintf("about to create CR %+v\n", cr))

	result, err := pipelineClient.Create(&cr)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	log.Info().Msg("created CR " + result.Name)

	return nil
}

func fillCRDefaults(cr *v1alpha1.Pipeline, pipelineName string) {

	cr.APIVersion = "churro.project.io/v1alpha1"
	cr.Kind = "Pipeline"
	cr.ObjectMeta.Name = pipelineName
	cr.Labels = make(map[string]string)
	cr.Labels["name"] = pipelineName
	cr.Status.Active = "true"
	cr.Status.Standby = []string{"one", "two"}

	/**
	cr.Spec.Functions = make([]v1alpha1.TransformFunction, 0)
	sample := v1alpha1.TransformFunction{}
	sample.ID = "one"
	sample.Name = "something"
	sample.Source = "some source"
	cr.Spec.Functions = append(cr.Spec.Functions, sample)

	cr.Spec.Extractsources = make([]v1alpha1.ExtractSourceDefinition, 0)
	sample2 := v1alpha1.ExtractSourceDefinition{}
	sample2.ID = "id21"
	sample2.Name = "extsource1"
	sample2.Path = "somepath"
	sample2.Scheme = "csv"
	sample2.Regex = "kjkjregex"
	sample2.Tablename = "mytable"
	sample2.Cronexpression = "@every 2m"
	cr.Spec.Extractsources = append(cr.Spec.Extractsources, sample2)

	cr.Spec.Extensions = make([]v1alpha1.ExtensionDefinition, 0)
	extsample := v1alpha1.ExtensionDefinition{}
	extsample.ID = "jkj"
	extsample.Extractsourceid = "kjk"
	extsample.Extensionname = "foo"
	extsample.Extensionpath = "somepath"
	cr.Spec.Extensions = append(cr.Spec.Extensions, extsample)

	cr.Spec.Extractrules = make([]v1alpha1.ExtractRuleDefinition, 0)
	rulesample := v1alpha1.ExtractRuleDefinition{}
	rulesample.ID = "jkjk"
	rulesample.ColumnName = "colname"
	rulesample.ColumnPath = "colpath"
	rulesample.MatchValues = "matchva"
	rulesample.TransformFunctionName = "fuinname"
	cr.Spec.Extractrules = append(cr.Spec.Extractrules, rulesample)
	*/

	cr.Spec.AdminDataSource.Name = "churrodatastore"
	cr.Spec.AdminDataSource.Scheme = ""
	cr.Spec.AdminDataSource.Username = "root"
	cr.Spec.AdminDataSource.Database = "churro"
	cr.Spec.DataSource.Name = "pipelinedatastore"
	cr.Spec.DataSource.Scheme = ""
	cr.Spec.DataSource.Username = pipelineName
	cr.Spec.DataSource.Database = pipelineName
	cr.Spec.WatchConfig.Location.Scheme = "http"
	cr.Spec.WatchConfig.Location.Host = "churro-watch"
	cr.Spec.WatchConfig.Location.Port = 8087

	switch cr.Spec.DatabaseType {
	case domain.DatabaseCockroach:
		cr.Spec.AdminDataSource.Host = "cockroachdb-public"
		cr.Spec.AdminDataSource.Path = ""
		cr.Spec.AdminDataSource.Port = 26257
		cr.Spec.DataSource.Host = "cockroachdb-public"
		cr.Spec.DataSource.Path = ""
		cr.Spec.DataSource.Port = 26257
	case domain.DatabaseMysql:
		cr.Spec.AdminDataSource.Host = "churro-pipeline-mysql-mysql"
		cr.Spec.AdminDataSource.Path = ""
		//cr.Spec.AdminDataSource.Password = "not-so-secure"
		cr.Spec.AdminDataSource.Port = 3306
		cr.Spec.DataSource.Host = "churro-pipeline-mysql-mysql"
		cr.Spec.DataSource.Path = ""
		//cr.Spec.DataSource.Password = "not-so-secure"
		cr.Spec.DataSource.Port = 3306
	case domain.DatabaseSinglestore:
		cr.Spec.AdminDataSource.Host = "svc-memsql-cluster-ddl"
		cr.Spec.AdminDataSource.Path = ""
		// TODO hard coded initial password for now
		cr.Spec.AdminDataSource.Password = "secretpass"
		cr.Spec.AdminDataSource.Port = 3306
		cr.Spec.DataSource.Host = "svc-memsql-cluster-ddl"
		cr.Spec.DataSource.Path = ""
		cr.Spec.DataSource.Password = "secretpass"
		cr.Spec.DataSource.Port = 3306
		cr.Spec.AdminDataSource.Username = "admin"
		cr.Spec.AdminDataSource.Database = "churro"
	}
}
