// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package extract

import (
	"context"
	"crypto/x509"

	"fmt"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/pipeline"
	pb "github.com/churrodata/churro/rpc/extension"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func (s *Server) processExtensions(elem extractapi.LoaderMessage) {
	extensions := s.ExtractSource.Extensions
	for _, ext := range extensions {
		log.Info().Msg(fmt.Sprintf("extension to process was  +%v", ext))
		extClient, err := GetExtensionServiceConnection(s.Pi.Name, ext.ExtensionPath)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in connecting to extension")
		}

		req := pb.PushRequest{
			Key:        elem.Key,
			Metadata:   elem.Metadata,
			DataFormat: elem.DataFormat,
		}

		_, err = extClient.Push(context.TODO(), &req)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in calling extension")
		}
	}
}

// GetExtensionServiceConnection ....
func GetExtensionServiceConnection(namespace, url string) (client pb.ExtensionClient, err error) {

	// get the service.crt from the pipeline CustomResource

	var p v1alpha1.Pipeline
	p, err = pipeline.GetPipeline(namespace)
	if err != nil {
		return client, err
	}

	serviceCrt1 := p.Spec.ServiceCredentials.ServiceCrt
	log.Info().Msg("got the ServiceCrt")

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(serviceCrt1))
	creds := credentials.NewClientTLSFromCert(caCertPool, "")

	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(creds))
	if err != nil {
		return client, err
	}

	client = pb.NewExtensionClient(conn)

	return client, nil

}
