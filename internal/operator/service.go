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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ctlService             = "churro-ctl"
	extractSourceService   = "churro-extractsource"
	cockroachService       = "cockroachdb"
	cockroachPublicService = "cockroachdb-public"
)

func (r PipelineReconciler) processServices(pipeline v1alpha1.Pipeline) error {
	// get referenced service objects
	var childServices v1.ServiceList
	err := r.List(r.Ctx, &childServices, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child Service")
		return err
	}

	// compare referenced Service objects with what we expect
	// make sure we have a service/churro-ctl
	needCtlService := true
	// make sure we have a service/churro-extractsource
	needExtractSourceService := true
	// make sure we have a service/cockroachdb-public
	needCockroachPublicService := true
	// make sure we have a service/cockroachdb
	needCockroachService := true

	for i := 0; i < len(childServices.Items); i++ {
		r := childServices.Items[i]
		switch r.Name {
		case ctlService:
			needCtlService = false
		case cockroachService:
			needCockroachService = false
		case extractSourceService:
			needExtractSourceService = false
		case cockroachPublicService:
			needCockroachPublicService = false
		}
	}

	// create any expected rbac objects, set owner reference to this pipeline
	servicesToCreate := make([]v1.Service, 0)
	if needCtlService {
		service := v1.Service{}
		service.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name, "service": "churro-ctl"}
		service.Name = ctlService
		service.Namespace = pipeline.ObjectMeta.Namespace
		sp := v1.ServicePort{
			Name: "grpc",
			Port: 8088,
		}
		service.Spec.Ports = []v1.ServicePort{sp}
		service.Spec.Selector = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name, "service": "churro-ctl"}
		/**
		  apiVersion: v1
		  kind: Service
		  metadata:
		    name: churro-ctl
		    labels:
		      app: churro
		      pipeline: pipeline1
		      service: churro-ctl
		  spec:
		    ports:
		    - port: 8088
		      targetPort: 8088
		      name: grpc
		    selector:
		      app: churro
		      pipeline: pipeline1
		      service: churro-ctl
		*/
		servicesToCreate = append(servicesToCreate, service)
	}
	if needExtractSourceService {
		service := v1.Service{}
		service.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name, "service": "churro-extractsource"}
		service.Name = extractSourceService
		service.Namespace = pipeline.ObjectMeta.Namespace
		// TODO replace hard-coded ports with constants
		sp := v1.ServicePort{
			Name: "grpc",
			Port: 8087,
		}
		service.Spec.Ports = []v1.ServicePort{sp}
		service.Spec.Selector = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name, "service": "churro-extractsource"}
		/**
		  apiVersion: v1
		  kind: Service
		  metadata:
		    name: churro-extractsource
		    labels:
		      app: churro
		      pipeline: pipeline1
		      service: churro-extractsource
		  spec:
		    ports:
		    - port: 8087
		      targetPort: 8087
		      name: grpc
		    selector:
		      app: churro
		      pipeline: pipeline1
		      service: churro-extractsource
		*/
		servicesToCreate = append(servicesToCreate, service)
	}
	if needCockroachService {
		service := v1.Service{}
		service.ObjectMeta.Labels = map[string]string{"app": "cockroachdb"}
		service.ObjectMeta.Annotations = map[string]string{"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true", "prometheus.io/scrape": "true", "prometheus.io/path": "_status/vars", "prometheus.io/port": "8080"}
		service.Name = cockroachService
		service.Namespace = pipeline.ObjectMeta.Namespace

		sp := v1.ServicePort{
			Name: "grpc",
			Port: 26257,
		}
		service.Spec.Ports = []v1.ServicePort{sp}
		sp = v1.ServicePort{
			Name: "http",
			Port: 8080,
		}
		service.Spec.Ports = append(service.Spec.Ports, sp)
		service.Spec.Selector = map[string]string{"app": "cockroachdb"}
		service.Spec.PublishNotReadyAddresses = true
		service.Spec.ClusterIP = "None"
		/**
		  apiVersion: v1
		  kind: Service
		  metadata:
		    name: cockroachdb
		    labels:
		      app: cockroachdb
		    annotations:
		      service.alpha.kubernetes.io/tolerate-unready-endpoints: "true"
		      prometheus.io/scrape: "true"
		      prometheus.io/path: "_status/vars"
		      prometheus.io/port: "8080"
		  spec:
		    ports:
		    - port: 26257
		      targetPort: 26257
		      name: grpc
		    - port: 8080
		      targetPort: 8080
		      name: http
		    publishNotReadyAddresses: true
		    clusterIP: None
		    selector:
		      app: cockroachdb
		*/
		servicesToCreate = append(servicesToCreate, service)
	}
	if needCockroachPublicService {
		service := v1.Service{}
		service.ObjectMeta.Labels = map[string]string{"app": "cockroachdb"}
		service.Name = cockroachPublicService
		service.Namespace = pipeline.ObjectMeta.Namespace

		/**
		  apiVersion: v1
		  kind: Service
		  metadata:
		    name: cockroachdb-public
		    labels:
		      app: cockroachdb
		  spec:
		    ports:
		    # The main port, served by gRPC, serves Postgres-flavor SQL, internode
		    # traffic and the cli.
		    - port: 26257
		      targetPort: 26257
		      name: grpc
		    # The secondary port serves the UI as well as health and debug endpoints.
		    - port: 8080
		      targetPort: 8080
		      name: http
		    selector:
		      app: cockroachdb
		*/
		sp := v1.ServicePort{
			Name: "grpc",
			Port: 26257,
		}
		service.Spec.Ports = []v1.ServicePort{sp}
		sp = v1.ServicePort{
			Name: "http",
			Port: 8080,
		}
		service.Spec.Selector = map[string]string{"app": "cockroachdb"}
		service.Spec.Ports = append(service.Spec.Ports, sp)
		servicesToCreate = append(servicesToCreate, service)
	}

	for _, svc := range servicesToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &svc, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &svc); err != nil {
			r.Log.Error(err, "unable to create Service for pipeline", "service", svc)
			return err
		}
		r.Log.V(1).Info("created Service for pipeline ", "service", svc)
	}

	return nil
}
