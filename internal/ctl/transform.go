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
	"encoding/json"
	"fmt"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateTransformFunction ...
func (s *Server) CreateTransformFunction(ctx context.Context, request *pb.CreateTransformFunctionRequest) (response *pb.CreateTransformFunctionResponse, err error) {

	response = &pb.CreateTransformFunctionResponse{}

	byt := []byte(request.FunctionString)
	var p domain.TransformFunction
	err = json.Unmarshal(byt, &p)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}

	if p.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"transform name is required")
	}
	if p.Source == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"transform source is required")
	}

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.AdminDataSource)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	tf := domain.TransformFunction{
		Name:   p.Name,
		Source: p.Source,
		ID:     xid.New().String(),
	}
	/**
	tf.ID, err = churroDB.CreateTransformFunction(tf)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	*/

	// create the transform function in the CR
	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return response, err
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return response, err
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return response, err
	}
	x := v1alpha1.TransformFunction{
		ID:     tf.ID,
		Name:   tf.Name,
		Source: tf.Source,
	}
	log.Info().Msg(fmt.Sprintf("adding transform %v", x))

	if len(pipelineToUpdate.Spec.Functions) == 0 {
		log.Info().Msg("allocating transformfunctions array")
		pipelineToUpdate.Spec.Functions = make([]v1alpha1.TransformFunction, 0)
	}
	pipelineToUpdate.Spec.Functions = append(pipelineToUpdate.Spec.Functions, x)

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		fmt.Printf("jeff here is an error2 %s\n", err.Error())
		log.Error().Stack().Err(err).Msg("some error")
		return response, err
	}

	response.ID = tf.ID
	return response, nil
}

// UpdateTransformFunction ...
func (s *Server) UpdateTransformFunction(ctx context.Context, request *pb.UpdateTransformFunctionRequest) (response *pb.UpdateTransformFunctionResponse, err error) {

	response = &pb.UpdateTransformFunctionResponse{}

	byt := []byte(request.FunctionString)
	var o domain.TransformFunction
	err = json.Unmarshal(byt, &o)
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

	if len(pipelineToUpdate.Spec.Functions) == 0 {
		log.Error().Msg("no transform functions exist")
		return nil, status.Errorf(codes.InvalidArgument, "no transform functions exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Functions); i++ {
		if pipelineToUpdate.Spec.Functions[i].ID == o.ID {
			pipelineToUpdate.Spec.Functions[i].Name = o.Name
			pipelineToUpdate.Spec.Functions[i].Source = o.Source

			_, err = pipelineClient.Update(pipelineToUpdate)
			if err != nil {
				log.Error().Stack().Err(err).Msg("some error")
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
		}
	}

	return response, nil
}

// GetTransformFunctions ...
func (s *Server) GetTransformFunctions(ctx context.Context, request *pb.GetTransformFunctionsRequest) (response *pb.GetTransformFunctionsResponse, err error) {

	response = &pb.GetTransformFunctionsResponse{}

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

	functions := make([]domain.TransformFunction, 0)
	for i := 0; i < len(pipelineToUpdate.Spec.Functions); i++ {
		fn := domain.TransformFunction{
			ID:     pipelineToUpdate.Spec.Functions[i].ID,
			Name:   pipelineToUpdate.Spec.Functions[i].Name,
			Source: pipelineToUpdate.Spec.Functions[i].Source,
		}
		functions = append(functions, fn)
	}

	b, _ := json.Marshal(functions)
	response.FunctionsString = string(b)

	return response, nil
}

// DeleteTransformFunction ...
func (s *Server) DeleteTransformFunction(ctx context.Context, request *pb.DeleteTransformFunctionRequest) (response *pb.DeleteTransformFunctionResponse, err error) {
	response = &pb.DeleteTransformFunctionResponse{}

	log.Info().Msg(fmt.Sprintf("deleting transform function ns=%s id=%s\n", request.Namespace, request.FunctionID))

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

	if len(pipelineToUpdate.Spec.Functions) == 0 {
		log.Error().Msg("no transform functions exist")
		return nil, status.Errorf(codes.InvalidArgument, "no transform functions exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Functions); i++ {
		if pipelineToUpdate.Spec.Functions[i].ID == request.FunctionID {
			// removes the transform function from the array
			pipelineToUpdate.Spec.Functions = append(pipelineToUpdate.Spec.Functions[:i], pipelineToUpdate.Spec.Functions[i+1:]...)
		}
	}

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return response, nil
}

// GetTransformFunction fetches a single transform function
func (s *Server) GetTransformFunction(ctx context.Context, request *pb.GetTransformFunctionRequest) (response *pb.GetTransformFunctionResponse, err error) {

	response = &pb.GetTransformFunctionResponse{}

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
	if len(pipelineToUpdate.Spec.Functions) == 0 {
		log.Error().Msg("no transform functions exist")
		return nil, status.Errorf(codes.InvalidArgument, "no transform functions exist")
	}

	tf := domain.TransformFunction{}
	for i := 0; i < len(pipelineToUpdate.Spec.Functions); i++ {
		if pipelineToUpdate.Spec.Functions[i].ID == request.FunctionID {
			tf.ID = pipelineToUpdate.Spec.Functions[i].ID
			tf.Name = pipelineToUpdate.Spec.Functions[i].Name
			tf.Source = pipelineToUpdate.Spec.Functions[i].Source
		}
	}

	b, _ := json.Marshal(tf)
	response.FunctionString = string(b)

	return response, nil
}
