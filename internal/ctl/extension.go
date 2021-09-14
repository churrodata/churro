// Copyright 2021 churrodata LLC
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
	"encoding/json"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateExtension ...
func (s *Server) CreateExtension(ctx context.Context, request *pb.CreateExtensionRequest) (response *pb.CreateExtensionResponse, err error) {

	log.Info().Msg("ctl.CreateExtension called here")
	response = &pb.CreateExtensionResponse{}
	var ext domain.Extension

	err = json.Unmarshal([]byte(request.ExtensionString), &ext)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}

	if ext.ExtractSourceID == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extension extract sourced ID is required")

	}
	if ext.ExtensionName == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extension name is required")
	}
	if ext.ExtensionPath == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extension path is required")
	}

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

	x := v1alpha1.ExtensionDefinition{
		ID:              ext.ID,
		Extractsourceid: ext.ExtractSourceID,
		Extensionname:   ext.ExtensionName,
		Extensionpath:   ext.ExtensionPath,
	}

	if len(pipelineToUpdate.Spec.Extensions) == 0 {
		pipelineToUpdate.Spec.Extensions = make([]v1alpha1.ExtensionDefinition, 0)
	}

	pipelineToUpdate.Spec.Extensions = append(pipelineToUpdate.Spec.Extensions, x)
	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	response.ID = ext.ID
	return response, nil
}

// DeleteExtension ...
func (s *Server) DeleteExtension(ctx context.Context, request *pb.DeleteExtensionRequest) (response *pb.DeleteExtensionResponse, err error) {

	response = &pb.DeleteExtensionResponse{}

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

	if len(pipelineToUpdate.Spec.Extensions) == 0 {
		log.Error().Msg("no extensions exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extensions exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extensions); i++ {
		if pipelineToUpdate.Spec.Extensions[i].ID == request.ExtensionID {
			// removes the extension from the array
			pipelineToUpdate.Spec.Extensions = append(pipelineToUpdate.Spec.Extensions[:i], pipelineToUpdate.Spec.Extensions[i+1:]...)
		}
	}

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return response, nil
}

// UpdateExtension ...
func (s *Server) UpdateExtension(ctx context.Context, request *pb.UpdateExtensionRequest) (response *pb.UpdateExtensionResponse, err error) {

	response = &pb.UpdateExtensionResponse{}

	var ext domain.Extension

	err = json.Unmarshal([]byte(request.ExtensionString), &ext)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}

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

	if len(pipelineToUpdate.Spec.Extensions) == 0 {
		log.Error().Msg("no extensions exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extensions exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extensions); i++ {
		if pipelineToUpdate.Spec.Extensions[i].ID == ext.ID {
			pipelineToUpdate.Spec.Extensions[i].Extensionname = ext.ExtensionName
			pipelineToUpdate.Spec.Extensions[i].Extensionpath = ext.ExtensionPath
			_, err = pipelineClient.Update(pipelineToUpdate)
			if err != nil {
				log.Error().Stack().Err(err).Msg("some error")
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
		}
	}

	return response, nil
}

// GetExtension ...
func (s *Server) GetExtension(ctx context.Context, request *pb.GetExtensionRequest) (response *pb.GetExtensionResponse, err error) {

	response = &pb.GetExtensionResponse{}

	var ext domain.Extension

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

	if len(pipelineToUpdate.Spec.Extensions) == 0 {
		log.Error().Msg("no extensions exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extensions exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extensions); i++ {
		if pipelineToUpdate.Spec.Extensions[i].ID == request.ExtensionID {
			ext.ID = pipelineToUpdate.Spec.Extensions[i].ID
			ext.ExtractSourceID = pipelineToUpdate.Spec.Extensions[i].Extractsourceid
			ext.ExtensionName = pipelineToUpdate.Spec.Extensions[i].Extensionname
			ext.ExtensionPath = pipelineToUpdate.Spec.Extensions[i].Extensionpath
			b, err := json.Marshal(ext)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			response.ExtensionString = string(b)

			return response, nil
		}
	}

	return nil, status.Errorf(codes.InvalidArgument, "extension not found")
}

// GetExtensions ...
func (s *Server) GetExtensions(ctx context.Context, request *pb.GetExtensionsRequest) (response *pb.GetExtensionsResponse, err error) {

	response = &pb.GetExtensionsResponse{}

	var exts []domain.Extension
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

	for i := 0; i < len(pipelineToUpdate.Spec.Extensions); i++ {
		ext := domain.Extension{}
		ext.ID = pipelineToUpdate.Spec.Extensions[i].ID
		ext.ExtractSourceID = pipelineToUpdate.Spec.Extensions[i].Extractsourceid
		ext.ExtensionName = pipelineToUpdate.Spec.Extensions[i].Extensionname
		ext.ExtensionPath = pipelineToUpdate.Spec.Extensions[i].Extensionpath
		exts = append(exts, ext)
	}

	b, err := json.Marshal(exts)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}
	response.ExtensionsString = string(b)

	return response, nil
}
