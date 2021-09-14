// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package ctl

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	"github.com/churrodata/churro/rpc/ctl"
	pb "github.com/churrodata/churro/rpc/ctl"
)

// GetPipeline gets information on a pipeline
func (s *Server) GetPipeline(ctx context.Context, req *ctl.GetPipelineRequest) (*pb.GetPipelineResponse, error) {

	resp := &pb.GetPipelineResponse{
		LoaderStatus:        "ok",
		ExtractsourceStatus: "ok",
		DatabaseConsoleURL:  "",
	}

	var err error

	resp.DatabaseConsoleURL, err = getDBConsoleURL(domain.DatabaseCockroach, s.Pi.Name)
	return resp, err
}

func getDBConsoleURL(databaseType, namespace string) (string, error) {
	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return "", err
	}

	var serviceName string
	if databaseType == domain.DatabaseCockroach {
		serviceName = "cockroachdb-public"
	} else {
		return "", errors.New("unsupported database")
	}

	getOptions := metav1.GetOptions{}
	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, getOptions)
	if err != nil {
		return "", err
	}

	scheme := "https://"
	port := ":8080"
	myURL := scheme + svc.Spec.ClusterIPs[0] + port

	if len(svc.Spec.ExternalIPs) > 0 {
		myURL = scheme + svc.Spec.ExternalIPs[0] + port
	} else if svc.Spec.LoadBalancerIP != "" {
		myURL = scheme + svc.Spec.LoadBalancerIP + port
	}

	return myURL, nil

}
