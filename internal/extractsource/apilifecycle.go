// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package extractsource

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/extractsource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StartAPI ...
func (s *Server) StartAPI(ctx context.Context, req *pb.StartAPIRequest) (response *pb.StartAPIResponse, err error) {

	resp := &pb.StartAPIResponse{}

	log.Info().Msg(fmt.Sprintf("StartAPIURL %+v\n", req))

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var wd domain.ExtractSource

	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		es := pipelineToUpdate.Spec.Extractsources[i]
		if req.ExtractSourceID == es.ID {
			wd = domain.ExtractSource{
				ID:             es.ID,
				Name:           es.Name,
				Path:           es.Path,
				Scheme:         es.Scheme,
				Regex:          es.Regex,
				Tablename:      es.Tablename,
				Cronexpression: es.Cronexpression,
				Skipheaders:    es.Skipheaders,
			}

		}
	}

	otherClient, err := GetKubeClient("")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting otherclient")
		return resp, err
	}

	err = s.createExtractPod(otherClient, wd.Scheme, wd.Path, s.Pi, wd.Tablename, wd.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating extract pod err")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return resp, nil
}

// StopAPI ...
func (s *Server) StopAPI(ctx context.Context, req *pb.StopAPIRequest) (response *pb.StopAPIResponse, err error) {

	resp := &pb.StopAPIResponse{}

	namespace := os.Getenv("CHURRO_NAMESPACE")
	if namespace == "" {
		log.Error().Stack().Msg("error CHURRO_NAMESPACE is empty")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	clientset, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting kubeclientset err ")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var wdName string
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if req.ExtractSourceID == pipelineToUpdate.Spec.Extractsources[i].ID {
			wdName = pipelineToUpdate.Spec.Extractsources[i].Name
		}
	}

	labelSelector := fmt.Sprintf("service=churro-extract,extractsourcename=%s", wdName)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error listing pods ")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if len(pods.Items) > 1 {
		return nil, status.Errorf(codes.InvalidArgument, "too many pods from list")
	}
	if len(pods.Items) < 1 {
		log.Info().Msg("no pod found to delete which can be ok in this case")
		return resp, nil
	}

	p := pods.Items[0]
	err = clientset.CoreV1().Pods(namespace).Delete(context.Background(), p.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Stack().Err(err).Msg("error deleting pod err")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return resp, nil
}
