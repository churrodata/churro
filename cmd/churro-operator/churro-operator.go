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
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/churrodata/churro/api/uiv1alpha1"
	churrov1alpha1 "github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/operator"
	mysqlv1alpha1 "github.com/presslabs/mysql-operator/pkg/apis"
	"github.com/rs/zerolog/log"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

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
	var err error
	path := fmt.Sprintf("%s%s%s", templatePath, "/", pvcFileName)
	flags.pvcTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", cockroachClientFileName)
	flags.cockroachClientPodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", singlestoreClientFileName)
	flags.singlestoreClientPodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", singlestoreClientServiceFileName)
	flags.singlestoreClientSvcTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", extractSourceFileName)
	flags.extractSourcePodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", ctlFileName)
	flags.ctlPodTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", statefulSetFileName)
	flags.statefulsetTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", mysqlCRFileName)
	flags.mysqlCRTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", mysqlSecretFileName)
	flags.mysqlSecretTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", singlestoreCRFileName)
	flags.singlestoreCRTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiDeploymentFileName)
	flags.uiDeploymentTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiPVCFileName)
	flags.uiPVCTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiSinglestoreOperatorFileName)
	flags.uiSinglestoreOperatorTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}
	path = fmt.Sprintf("%s%s%s", templatePath, "/", uiCockroachDBOperatorFileName)
	flags.uiCockroachDBOperatorTemplate, err = ioutil.ReadFile(path)
	if err != nil {
		setupLog.Error(err, "unable to read "+path)
		os.Exit(1)
	}

	return flags
}
