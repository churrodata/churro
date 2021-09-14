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
	"errors"
	"fmt"
	"strconv"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/ohler55/ojg/jp"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/xmlpath.v2"
)

// CreateExtractRule ....
func (s *Server) CreateExtractRule(ctx context.Context, request *pb.CreateExtractRuleRequest) (response *pb.CreateExtractRuleResponse, err error) {

	response = &pb.CreateExtractRuleResponse{}
	var rule domain.ExtractRule

	err = json.Unmarshal([]byte(request.ExtractRuleString), &rule)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}

	if rule.ExtractSourceID == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract rule extract source ID is required")

	}
	if rule.ColumnName == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract rule column name is required")
	}
	if rule.ColumnPath == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract rule source is required")
	}
	if rule.ColumnType == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract rule type is required")
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

	wdir := domain.ExtractSource{
		ID: rule.ExtractSourceID,
	}
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if rule.ExtractSourceID == pipelineToUpdate.Spec.Extractsources[i].ID {
			wdir.Scheme = pipelineToUpdate.Spec.Extractsources[i].Scheme
		}
	}

	err = validateRulePath(rule.ColumnPath, wdir.Scheme)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	rule.ID = xid.New().String()

	if len(pipelineToUpdate.Spec.Extractrules) == 0 {
		log.Info().Msg("allocating transformfunctions array")
		pipelineToUpdate.Spec.Extractrules = make([]v1alpha1.ExtractRuleDefinition, 0)
	}
	x := v1alpha1.ExtractRuleDefinition{
		ID:                    rule.ID,
		Extractsourceid:       rule.ExtractSourceID,
		ColumnName:            rule.ColumnName,
		ColumnPath:            rule.ColumnPath,
		ColumnType:            rule.ColumnType,
		MatchValues:           rule.MatchValues,
		TransformFunctionName: rule.TransformFunction,
	}
	pipelineToUpdate.Spec.Extractrules = append(pipelineToUpdate.Spec.Extractrules, x)

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	response.ID = rule.ID
	return response, nil
}

// DeleteExtractRule deletes an extract rule
func (s *Server) DeleteExtractRule(ctx context.Context, request *pb.DeleteExtractRuleRequest) (response *pb.DeleteExtractRuleResponse, err error) {

	response = &pb.DeleteExtractRuleResponse{}

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

	if len(pipelineToUpdate.Spec.Extractrules) == 0 {
		log.Error().Msg("no extract rules exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extract rules exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extractrules); i++ {
		if pipelineToUpdate.Spec.Extractrules[i].ID == request.ExtractRuleID {
			// removes the extract rule from the array
			pipelineToUpdate.Spec.Extractrules = append(pipelineToUpdate.Spec.Extractrules[:i], pipelineToUpdate.Spec.Extractrules[i+1:]...)
		}
	}

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return response, nil
}

// UpdateExtractRule updates an extract rule
func (s *Server) UpdateExtractRule(ctx context.Context, request *pb.UpdateExtractRuleRequest) (response *pb.UpdateExtractRuleResponse, err error) {

	response = &pb.UpdateExtractRuleResponse{}

	var rule domain.ExtractRule

	err = json.Unmarshal([]byte(request.ExtractRuleString), &rule)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}

	log.Info().Msg(fmt.Sprintf("in updateextractrule got rule %+v\n", rule))

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
	wdir := domain.ExtractSource{
		ID: rule.ExtractSourceID,
	}
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if pipelineToUpdate.Spec.Extractsources[i].ID == rule.ExtractSourceID {
			wdir.Scheme = pipelineToUpdate.Spec.Extractsources[i].Scheme
		}
	}

	err = validateRulePath(rule.ColumnPath, wdir.Scheme)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if len(pipelineToUpdate.Spec.Extractrules) == 0 {
		log.Error().Msg("no extract rules exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extract rules exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extractrules); i++ {
		if pipelineToUpdate.Spec.Extractrules[i].ID == rule.ID {
			pipelineToUpdate.Spec.Extractrules[i].ColumnName = rule.ColumnName
			pipelineToUpdate.Spec.Extractrules[i].ColumnPath = rule.ColumnPath
			pipelineToUpdate.Spec.Extractrules[i].MatchValues = rule.MatchValues
			pipelineToUpdate.Spec.Extractrules[i].TransformFunctionName = rule.TransformFunction
			_, err = pipelineClient.Update(pipelineToUpdate)
			if err != nil {
				log.Error().Stack().Err(err).Msg("some error")
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
		}
	}

	return response, nil
}

// GetExtractRule fetches an extract rule
func (s *Server) GetExtractRule(ctx context.Context, request *pb.GetExtractRuleRequest) (response *pb.GetExtractRuleResponse, err error) {

	response = &pb.GetExtractRuleResponse{}

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

	//var rule domain.ExtractRule
	for i := 0; i < len(pipelineToUpdate.Spec.Extractrules); i++ {
		if pipelineToUpdate.Spec.Extractrules[i].ID == request.ExtractRuleID {
			rule := domain.ExtractRule{
				ID:                request.ExtractRuleID,
				ExtractSourceID:   pipelineToUpdate.Spec.Extractrules[i].Extractsourceid,
				ColumnName:        pipelineToUpdate.Spec.Extractrules[i].ColumnName,
				ColumnPath:        pipelineToUpdate.Spec.Extractrules[i].ColumnPath,
				ColumnType:        pipelineToUpdate.Spec.Extractrules[i].ColumnType,
				MatchValues:       pipelineToUpdate.Spec.Extractrules[i].MatchValues,
				TransformFunction: pipelineToUpdate.Spec.Extractrules[i].TransformFunctionName,
			}

			b, err := json.Marshal(rule)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			response.ExtractRuleString = string(b)

			return response, nil
		}
	}
	return nil, status.Errorf(codes.InvalidArgument, err.Error())

}

// GetExtractRules fetches all extract rules
func (s *Server) GetExtractRules(ctx context.Context, request *pb.GetExtractRulesRequest) (response *pb.GetExtractRulesResponse, err error) {

	response = &pb.GetExtractRulesResponse{}

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

	rules := make([]domain.ExtractRule, 0)
	for i := 0; i < len(pipelineToUpdate.Spec.Extractrules); i++ {
		rule := domain.ExtractRule{
			ID:                pipelineToUpdate.Spec.Extractrules[i].ID,
			ExtractSourceID:   pipelineToUpdate.Spec.Extractrules[i].Extractsourceid,
			ColumnName:        pipelineToUpdate.Spec.Extractrules[i].ColumnName,
			ColumnPath:        pipelineToUpdate.Spec.Extractrules[i].ColumnPath,
			ColumnType:        pipelineToUpdate.Spec.Extractrules[i].ColumnType,
			MatchValues:       pipelineToUpdate.Spec.Extractrules[i].MatchValues,
			TransformFunction: pipelineToUpdate.Spec.Extractrules[i].TransformFunctionName,
		}
		rules = append(rules, rule)
	}

	log.Info().Msg(fmt.Sprintf("returning values len %d\n", len(rules)))

	b, err := json.Marshal(rules)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}
	response.ExtractRulesString = string(b)

	return response, nil
}

func validateRulePath(path, scheme string) (err error) {
	switch scheme {
	case extractapi.XMLScheme:
		_, err = xmlpath.Compile(path)
	case extractapi.CSVScheme:
		// must be a positive integer to represent an integer
		var v int
		v, err = strconv.Atoi(path)
		if err != nil {
			return err
		}
		if v < 0 {
			err = errors.New("CSV path can not be a negative number")
		}
	case extractapi.JSONPathScheme:
		_, err = jp.ParseString(path)
	}
	return err
}
