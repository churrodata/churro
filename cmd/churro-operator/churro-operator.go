// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/churrodata/churro/api/uiv1alpha1"
	churrov1alpha1 "github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/operator"
	"github.com/churrodata/churro/pkg"
	mysqlv1alpha1 "github.com/presslabs/mysql-operator/pkg/apis"
	"github.com/rs/zerolog/log"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

//go:embed deploy/templates/*
var embeddedTemplates embed.FS

type ChurroOperatorFlags struct {
	metricsAddr                   string
	extractSourcePodTemplate      []byte
	pvcTemplate                   []byte
	statefulsetTemplate           []byte
	singlestoreCRTemplate         []byte
	mysqlCRTemplate               []byte
	mysqlSecretTemplate           []byte
	cockroachClientPodTemplate    []byte
	singlestoreClientPodTemplate  []byte
	singlestoreClientSvcTemplate  []byte
	ctlPodTemplate                []byte
	uiDeploymentTemplate          []byte
	uiPVCTemplate                 []byte
	uiCockroachDBCRDTemplate      []byte
	uiCockroachDBOperatorTemplate []byte
	uiSinglestoreCRDTemplate      []byte
	uiSinglestoreOperatorTemplate []byte
	enableLeaderElection          bool
}

const (
	templatePath                     = "/templates"
	cockroachClientFileName          = "client.yaml"
	singlestoreClientFileName        = "memsql-studio.yaml"
	singlestoreClientServiceFileName = "memsql-studio-svc.yaml"
	pvcFileName                      = "churrodata-pvc.yaml"
	statefulSetFileName              = "cockroachdb-statefulset.yaml"
	mysqlCRFileName                  = "mysql-cr.yaml"
	singlestoreCRFileName            = "singlestore-cr.yaml"
	mysqlSecretFileName              = "mysql-secret.yaml"
	extractSourceFileName            = "churro-extractsource.yaml"
	ctlFileName                      = "churro-ctl.yaml"
	uiDeploymentFileName             = "churro-ui.yaml"
	uiPVCFileName                    = "admindb-pvc.yaml"
	uiCockroachDBCRDFileName         = "cockroachdb-crd.yaml"
	uiSinglestoreCRDFileName         = "singlestore-crd.yaml"
	uiSinglestoreOperatorFileName    = "singlestore-operator.yaml"
	uiCockroachDBOperatorFileName    = "cockroachdb-operator.yaml"
	configMapName                    = "churro-templates"
	deployTemplatesDir               = "deploy/templates/"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = churrov1alpha1.AddToScheme(scheme)
	_ = uiv1alpha1.AddToScheme(scheme)
	_ = mysqlv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {

	flags := processFlags()

	log.Info().Msg("CHURRO_PULL_SECRET_NAME is " + os.Getenv("CHURRO_PULL_SECRET_NAME"))

	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: flags.metricsAddr,
		Port:               9443,
		LeaderElection:     flags.enableLeaderElection,
		LeaderElectionID:   "d296171c.project.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&operator.PipelineReconciler{
		Client:                       mgr.GetClient(),
		Log:                          ctrl.Log.WithName("controllers").WithName("Pipeline"),
		Scheme:                       mgr.GetScheme(),
		PVCTemplate:                  flags.pvcTemplate,
		ExtractSourcePodTemplate:     flags.extractSourcePodTemplate,
		CockroachClientPodTemplate:   flags.cockroachClientPodTemplate,
		SinglestoreClientPodTemplate: flags.singlestoreClientPodTemplate,
		SinglestoreClientSvcTemplate: flags.singlestoreClientSvcTemplate,
		CtlPodTemplate:               flags.ctlPodTemplate,
		StatefulSetTemplate:          flags.statefulsetTemplate,
		MYSQLCRTemplate:              flags.mysqlCRTemplate,
		MYSQLSecretTemplate:          flags.mysqlSecretTemplate,
		SinglestoreCRTemplate:        flags.singlestoreCRTemplate,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pipeline")
		os.Exit(1)
	}

	if err = (&operator.ChurrouiReconciler{
		Client:                      mgr.GetClient(),
		Log:                         ctrl.Log.WithName("controllers").WithName("Churroui"),
		Scheme:                      mgr.GetScheme(),
		UIDeploymentTemplate:        flags.uiDeploymentTemplate,
		PVCTemplate:                 flags.uiPVCTemplate,
		CockroachDBCRDTemplate:      flags.uiCockroachDBCRDTemplate,
		CockroachDBOperatorTemplate: flags.uiCockroachDBOperatorTemplate,
		SinglestoreCRDTemplate:      flags.uiSinglestoreCRDTemplate,
		SinglestoreOperatorTemplate: flags.uiSinglestoreOperatorTemplate,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create churroUI controller", "controller", "Churroui")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func processFlags() ChurroOperatorFlags {
	flags := ChurroOperatorFlags{}

	flag.StringVar(&flags.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&flags.enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	setupLog := ctrl.Log.WithName("setup")
	/**
	var err error
	path := fmt.Sprintf("%s%s%s", templatePath, "/", pvcFileName)
	flags.pvcTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.pvcTemplate = getTemplate(pvcFileName)
	setupLog.Info("found " + pvcFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", cockroachClientFileName)
	flags.cockroachClientPodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.cockroachClientPodTemplate = getTemplate(cockroachClientFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", singlestoreClientFileName)
	flags.singlestoreClientPodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.singlestoreClientPodTemplate = getTemplate(singlestoreClientFileName)
	/**

	path = fmt.Sprintf("%s%s%s", templatePath, "/", singlestoreClientServiceFileName)
	flags.singlestoreClientSvcTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.singlestoreClientSvcTemplate = getTemplate(singlestoreClientServiceFileName)
	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", extractSourceFileName)
	flags.extractSourcePodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.extractSourcePodTemplate = getTemplate(extractSourceFileName)
	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", ctlFileName)
	flags.ctlPodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.ctlPodTemplate = getTemplate(ctlFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", statefulSetFileName)
	flags.statefulsetTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.statefulsetTemplate = getTemplate(statefulSetFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", mysqlCRFileName)
	flags.mysqlCRTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.mysqlCRTemplate = getTemplate(mysqlCRFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", mysqlSecretFileName)
	flags.mysqlSecretTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.mysqlSecretTemplate = getTemplate(mysqlSecretFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", singlestoreCRFileName)
	flags.singlestoreCRTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.singlestoreCRTemplate = getTemplate(singlestoreCRFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiDeploymentFileName)
	flags.uiDeploymentTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.uiDeploymentTemplate = getTemplate(uiDeploymentFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiPVCFileName)
	flags.uiPVCTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.uiPVCTemplate = getTemplate(uiPVCFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiSinglestoreOperatorFileName)
	flags.uiSinglestoreOperatorTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.uiSinglestoreOperatorTemplate = getTemplate(uiSinglestoreOperatorFileName)

	/**
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiCockroachDBOperatorFileName)
	flags.uiCockroachDBOperatorTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	*/
	flags.uiCockroachDBOperatorTemplate = getTemplate(uiCockroachDBOperatorFileName)

	return flags
}

//getTemplate returns a template either from the ConfigMap if present or
// from the embedded data if the Configmap is not present, an empty
// template will cause the program to exit abnormally
func getTemplate(templateKey string) (templateBytes []byte) {

	ns := os.Getenv("CHURRO_NAMESPACE")


	// get the template from the embedded dir as a default
	data, err := embeddedTemplates.ReadFile(deployTemplatesDir + templateKey)
	if err != nil {
		fmt.Printf("error here %s\n", err.Error())
		os.Exit(2)
	}
	if len(data) == 0 {
		fmt.Printf("error here 2 %s\n", err.Error())
		os.Exit(2)
	}

	// connect to the Kube API
	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		fmt.Printf("error here 3 %s\n", err.Error())

		os.Exit(2)
	}

	fmt.Printf("looking for template key [%s]\n", templateKey)
	var thisMap *v1.ConfigMap
	thisMap, err = getConfigMap(clientset, ns, configMapName)
	if kerrors.IsNotFound(err) {

		fmt.Printf("configMap not found, will use embedded %s\n", templateKey)
		return data
	} else if err != nil {
		fmt.Printf("error here 4 %s\n", err.Error())

		os.Exit(2)
	}
	fmt.Printf("configmap found\n")
	str := thisMap.Data[templateKey]
	templateBytes = []byte(str)
	if len(templateBytes) == 0 {

		fmt.Printf("invalid template %s from ConfigMap length is 0, will use embedded version instead", templateKey)
		return data


	}
	fmt.Printf("found templateKey in ConfigMap! %s\n", templateBytes)
	return templateBytes
}

// getConfigMap method
func getConfigMap(clientset *kubernetes.Clientset, ns, configMapName string) (*v1.ConfigMap, error) {
	configMap, err := clientset.CoreV1().ConfigMaps(ns).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		//fmt.Printf("Error occured while creating configmap %s: %s", configMapName, err.Error())
		return nil, err
	}

	fmt.Printf("configMap %s is succesfully got", configMap.Name)
	return configMap, nil
}
