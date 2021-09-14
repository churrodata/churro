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

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/extractsource"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	watchpb "github.com/churrodata/churro/rpc/extractsource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
)

// CreateExtractSource ...
func (s *Server) CreateExtractSource(ctx context.Context, request *pb.CreateExtractSourceRequest) (response *pb.CreateExtractSourceResponse, err error) {

	response = &pb.CreateExtractSourceResponse{}

	var wdir domain.ExtractSource

	err = json.Unmarshal([]byte(request.ExtractSourceString), &wdir)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}

	if wdir.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract source name is required")

	}
	if wdir.Path == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract source path is required")
	}
	if wdir.Scheme == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract source scheme is required")
	}
	if wdir.Regex == "" && wdir.Scheme != extractapi.APIScheme {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract source regex is required")
	}
	if wdir.Skipheaders <= 0 && (wdir.Scheme == extractapi.APIScheme || wdir.Scheme == extractapi.XLSXScheme) {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract source skipheaders is required to be >= 0")
	}
	if wdir.Tablename == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"extract source tablename is required")
	}

	wdir.ID = xid.New().String()

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

	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if pipelineToUpdate.Spec.Extractsources[i].Path == wdir.Path {
			return nil, status.Errorf(codes.InvalidArgument, "path already taken")
		}
	}

	log.Info().Msg(fmt.Sprintf("create extract source id=%s for ns=%s\n", wdir.ID, request.Namespace))

	if len(pipelineToUpdate.Spec.Extractsources) == 0 {
		pipelineToUpdate.Spec.Extractsources = make([]v1alpha1.ExtractSourceDefinition, 0)
	}
	esrc := v1alpha1.ExtractSourceDefinition{
		ID:             wdir.ID,
		Name:           wdir.Name,
		Path:           wdir.Path,
		Scheme:         wdir.Scheme,
		Regex:          wdir.Regex,
		Tablename:      wdir.Tablename,
		Cronexpression: wdir.Cronexpression,
		Skipheaders:    wdir.Skipheaders,
	}

	pipelineToUpdate.Spec.Extractsources = append(pipelineToUpdate.Spec.Extractsources, esrc)

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	response.ID = wdir.ID

	err = s.triggerExtractSourceUpdate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return response, nil
}

// DeleteExtractSource ...
func (s *Server) DeleteExtractSource(ctx context.Context, request *pb.DeleteExtractSourceRequest) (response *pb.DeleteExtractSourceResponse, err error) {

	response = &pb.DeleteExtractSourceResponse{}

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

	if len(pipelineToUpdate.Spec.Extractsources) == 0 {
		log.Error().Msg("no extract sources exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extract sources exist")
	}

	var currentName string
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if pipelineToUpdate.Spec.Extractsources[i].ID == request.ExtractSourceID {
			// removes the extract source from the array
			currentName = pipelineToUpdate.Spec.Extractsources[i].Name
			pipelineToUpdate.Spec.Extractsources = append(pipelineToUpdate.Spec.Extractsources[:i], pipelineToUpdate.Spec.Extractsources[i+1:]...)
		}
	}

	_, err = pipelineClient.Update(pipelineToUpdate)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = s.triggerExtractSourceUpdate()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = deleteDataSourcePods(request.Namespace, currentName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return response, nil
}

// GetExtractSource ...
func (s *Server) GetExtractSource(ctx context.Context, request *pb.GetExtractSourceRequest) (response *pb.GetExtractSourceResponse, err error) {

	response = &pb.GetExtractSourceResponse{}

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

	wdir := domain.ExtractSource{}
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if pipelineToUpdate.Spec.Extractsources[i].ID == request.ExtractSourceID {
			c := pipelineToUpdate.Spec.Extractsources[i]
			wdir.ID = c.ID
			wdir.Name = c.Name
			wdir.Path = c.Path
			wdir.Scheme = c.Scheme
			wdir.Regex = c.Regex
			wdir.Tablename = c.Tablename
			wdir.Cronexpression = pipelineToUpdate.Spec.Extractsources[i].Cronexpression
			// get the extract rules for this extract source
			wdir.ExtractRules = make(map[string]domain.ExtractRule)
			for z := 0; z < len(pipelineToUpdate.Spec.Extractrules); z++ {
				if pipelineToUpdate.Spec.Extractrules[z].Extractsourceid == request.ExtractSourceID {
					a := pipelineToUpdate.Spec.Extractrules[z]
					dom := domain.ExtractRule{}
					dom.ID = a.ID
					dom.ExtractSourceID = a.Extractsourceid
					dom.ColumnName = a.ColumnName
					dom.ColumnPath = a.ColumnPath
					dom.ColumnType = a.ColumnType
					dom.MatchValues = a.MatchValues
					dom.TransformFunction = a.TransformFunctionName
					wdir.ExtractRules[a.ID] = dom
				}
			}

			// get the extensions for this extract source
			wdir.Extensions = make(map[string]domain.Extension)
			for y := 0; y < len(pipelineToUpdate.Spec.Extensions); y++ {
				if pipelineToUpdate.Spec.Extensions[y].Extractsourceid == request.ExtractSourceID {
					a := pipelineToUpdate.Spec.Extensions[y]
					dom := domain.Extension{}
					dom.ID = a.ID
					dom.ExtractSourceID = a.Extractsourceid
					dom.ExtensionName = a.Extensionname
					dom.ExtensionPath = a.Extensionpath
					wdir.Extensions[a.ID] = dom
				}
			}
		}
	}

	if wdir.Scheme == extractapi.APIScheme {
		var err error
		wdir.Running, err = isRunning(s.Pi.Name, wdir.Name)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}

	b, err := json.Marshal(wdir)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}
	response.ExtractSourceString = string(b)
	metrics, err := s.getExtractSourceMetrics(request.ExtractSourceID)
	response.Metrics = metrics

	return response, nil
}

// GetExtractSources ...
func (s *Server) GetExtractSources(ctx context.Context, request *pb.GetExtractSourcesRequest) (response *pb.GetExtractSourcesResponse, err error) {

	response = &pb.GetExtractSourcesResponse{}
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

	values := make([]domain.ExtractSource, 0)
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		current := pipelineToUpdate.Spec.Extractsources[i]
		v := domain.ExtractSource{
			ID:             current.ID,
			Name:           current.Name,
			Path:           current.Path,
			Scheme:         current.Scheme,
			Regex:          current.Regex,
			Tablename:      current.Tablename,
			Cronexpression: current.Cronexpression,
		}
		values = append(values, v)
	}

	log.Info().Msg(fmt.Sprintf("returning values len %d\n", len(values)))

	b, err := json.Marshal(values)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			err.Error())
	}
	response.ExtractSourcesString = string(b)

	return response, nil
}

// UpdateExtractSource ...
func (s *Server) UpdateExtractSource(ctx context.Context, request *pb.UpdateExtractSourceRequest) (response *pb.UpdateExtractSourceResponse, err error) {

	response = &pb.UpdateExtractSourceResponse{}

	var f domain.ExtractSource

	err = json.Unmarshal([]byte(request.ExtractSourceString), &f)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
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

	var current v1alpha1.ExtractSourceDefinition
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if f.ID == pipelineToUpdate.Spec.Extractsources[i].ID {
			current = pipelineToUpdate.Spec.Extractsources[i]
		}
	}

	if current.Path != f.Path {

		available := true
		for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
			if f.Path == pipelineToUpdate.Spec.Extractsources[i].Path {
				available = false
			}
		}
		if !available {
			return nil, status.Errorf(codes.InvalidArgument, "path already taken")
		}
	} else {
		log.Info().Msg("extractsource dir paths did not change in update")
	}

	if len(pipelineToUpdate.Spec.Extractsources) == 0 {
		log.Error().Msg("no extract sources exist")
		return nil, status.Errorf(codes.InvalidArgument, "no extract sources exist")
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if pipelineToUpdate.Spec.Extractsources[i].ID == f.ID {
			pipelineToUpdate.Spec.Extractsources[i].Name = f.Name
			pipelineToUpdate.Spec.Extractsources[i].Path = f.Path
			pipelineToUpdate.Spec.Extractsources[i].Scheme = f.Scheme
			pipelineToUpdate.Spec.Extractsources[i].Regex = f.Regex
			pipelineToUpdate.Spec.Extractsources[i].Tablename = f.Tablename
			pipelineToUpdate.Spec.Extractsources[i].Cronexpression = f.Cronexpression
			_, err = pipelineClient.Update(pipelineToUpdate)
			if err != nil {
				log.Error().Stack().Err(err).Msg("some error")
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
		}
	}

	err = s.triggerExtractSourceUpdate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return response, nil
}

func (s Server) triggerExtractSourceUpdate() error {
	// here is where we would ping the extractsource service to let it update

	client, err := extractsource.GetExtractSourceServiceConnection(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	req := watchpb.CreateExtractSourceRequest{}
	resp, err := client.CreateExtractSource(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	log.Info().Msg(fmt.Sprintf("create extract source resp %+v\n", resp))
	return nil
}

func (s Server) getExtractSourceMetrics(wdirid string) (metrics []*pb.PipelineMetric, err error) {
	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return metrics, err
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.AdminDataSource)
	if err != nil {
		return metrics, err
	}

	m, err := churroDB.GetExtractSourceMetrics(wdirid)
	if err != nil {
		return metrics, err
	}

	for i := 0; i < len(m); i++ {
		m1 := pb.PipelineMetric{
			Name:  m[i].Name,
			Value: m[i].Value,
		}
		metrics = append(metrics, &m1)
	}

	return metrics, nil
}

func isRunning(namespace, name string) (bool, error) {
	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return false, err
	}

	labelSelector := fmt.Sprintf("extractsourcename=%s", name)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return false, err
	}
	for i := 0; i < len(pods.Items); i++ {
		p := pods.Items[i]
		if p.Status.Phase == v1.PodRunning {
			return true, nil
		}
	}

	return false, nil
}

func deleteDataSourcePods(namespace, name string) error {
	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return err
	}

	labelSelector := fmt.Sprintf("extractsourcename=%s", name)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	delOptions := metav1.DeleteOptions{}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	for i := 0; i < len(pods.Items); i++ {
		p := pods.Items[i]
		log.Info().Msg("deleting the following pod " + p.Name)
		err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), p.Name, delOptions)
		if err != nil {
			return err
		}
		log.Info().Msg("deleted pod " + p.Name)
	}

	return nil
}
