// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package handlers

import (
	"crypto/x509"
	"fmt"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/ctl"
	"github.com/churrodata/churro/internal/pipeline"
	pb "github.com/churrodata/churro/rpc/ctl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GetServiceConnection ...
func GetServiceConnection(namespace string) (client pb.CtlClient, err error) {

	// get the service.crt from the pipeline CustomResource

	var p v1alpha1.Pipeline
	p, err = pipeline.GetPipeline(namespace)
	if err != nil {
		return client, err
	}

	serviceCrt1 := p.Spec.ServiceCredentials.ServiceCrt

	url := fmt.Sprintf("churro-ctl.%s.svc.cluster.local%s", namespace, ctl.DefaultPort)

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(serviceCrt1))
	creds := credentials.NewClientTLSFromCert(caCertPool, "")

	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(creds))
	if err != nil {
		return client, err
	}

	client = pb.NewCtlClient(conn)

	return client, nil

}
