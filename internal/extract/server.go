// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package extract holds the churro extract service implementation
package extract

import (
	"context"
	"fmt"
	"os"
	"strconv"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	"github.com/churrodata/churro/pkg/config"
	pb "github.com/churrodata/churro/rpc/extract"
	"github.com/rs/zerolog/log"
)

var sleepTime = 3 //seconds to sleep when backpressure
var backPressure int32

// Server is the extract server configuration
type Server struct {
	Pi                 v1alpha1.Pipeline
	ServiceCreds       config.ServiceCredentials
	DBCreds            config.DBCredentials
	TableName          string
	SchemeValue        string
	FileName           string
	TransformFunctions []domain.TransformFunction
	ExtractSource      domain.ExtractSource
	APIStopTime        int
}

// NewExtractServer creates an extract server based on the configPath
// and returns a pointer to the extract server
func NewExtractServer(fileName, schemeValue, tableName string, debug bool, svcCreds config.ServiceCredentials, dbCreds config.DBCredentials, pipeline v1alpha1.Pipeline) *Server {
	s := &Server{
		ServiceCreds: svcCreds,
		DBCreds:      dbCreds,
		Pi:           pipeline,
		FileName:     fileName,
		SchemeValue:  schemeValue,
		TableName:    tableName,
	}

	var err error
	var churroDB db.ChurroDatabase

	churroDB, err = db.NewChurroDB(pipeline.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not create the database")
		os.Exit(1)
	}

	err = churroDB.GetConnection(dbCreds, pipeline.Spec.AdminDataSource)

	if err != nil {
		log.Error().Stack().Err(err).Msg("could not open the database: ")
		os.Exit(1)
	}

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err)
		os.Exit(1)
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err)
		os.Exit(1)
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err)
		os.Exit(1)
	}

	s.TransformFunctions = make([]domain.TransformFunction, 0)

	for i := 0; i < len(pipelineToUpdate.Spec.Functions); i++ {
		fn := domain.TransformFunction{
			ID:     pipelineToUpdate.Spec.Functions[i].ID,
			Name:   pipelineToUpdate.Spec.Functions[i].Name,
			Source: pipelineToUpdate.Spec.Functions[i].Source,
		}
		s.TransformFunctions = append(s.TransformFunctions, fn)
	}

	//s.TransformFunctions, err = churroDB.GetTransformFunctions()

	log.Info().Msg(fmt.Sprintf("transform functions %d\n", len(s.TransformFunctions)))

	wdirName := os.Getenv("CHURRO_WATCHDIR_NAME")

	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		c := pipelineToUpdate.Spec.Extractsources[i]
		if c.Name == wdirName {
			s.ExtractSource = domain.ExtractSource{
				Name:           c.Name,
				ID:             c.ID,
				Path:           c.Path,
				Scheme:         c.Scheme,
				Regex:          c.Regex,
				Tablename:      c.Tablename,
				Cronexpression: c.Cronexpression,
				Skipheaders:    c.Skipheaders,
				Sheetname:      c.Sheetname,
				ExtractRules:   make(map[string]domain.ExtractRule),
			}
			g := pipelineToUpdate.Spec.Extractrules
			for i := 0; i < len(g); i++ {
				if g[i].Extractsourceid == c.ID {
					d := domain.ExtractRule{
						ID:              g[i].ID,
						ExtractSourceID: g[i].Extractsourceid,
						ColumnName:      g[i].ColumnName,
						ColumnPath:      g[i].ColumnPath,
						ColumnType:      g[i].ColumnType,
						MatchValues:     g[i].MatchValues,
					}
					d.TransformFunction = g[i].TransformFunctionName
					s.ExtractSource.ExtractRules[d.ID] = d
				}
			}
			s.ExtractSource.Extensions = make(map[string]domain.Extension)
			h := pipelineToUpdate.Spec.Extensions
			for i := 0; i < len(h); i++ {
				if h[i].Extractsourceid == c.ID {
					d := domain.Extension{
						ID:              h[i].ID,
						ExtractSourceID: h[i].Extractsourceid,
						ExtensionName:   h[i].Extensionname,
						ExtensionPath:   h[i].Extensionpath,
					}
					s.ExtractSource.Extensions[d.ID] = d
				}
			}
		}
	}

	ctx := context.Background()

	s.createMetric()

	log.Debug().Msg("NewExtractServer called processing started..." + schemeValue)
	switch schemeValue {
	case extractapi.HTTPPostScheme:
		log.Info().Msg("Info: extract is processing a httppost config")
		err = s.ExtractHTTPPost(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in httppost processing")
		}
	case extractapi.APIScheme:
		log.Info().Msg("Info: extract is processing a API config")
		err = s.ExtractAPI(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in JSON Stream processing")
		}
	case extractapi.XMLScheme:
		log.Info().Msg("Info: extract is processing a xml file")
		err = s.ExtractXML(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in xml processing")
		}
	case extractapi.CSVScheme:
		log.Info().Msg("Info: extract is processing a CSV file")
		err = s.ExtractCSV(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in csv processing")
		}
	case extractapi.XLSXScheme:
		log.Info().Msg("Info: extract is processing a xlsx file")
		err = s.ExtractXLS(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in xlsx processing ")
		}
	case extractapi.JSONScheme:
		log.Info().Msg("extract is processing a json file")
		err = s.ExtractJSON(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in json processing ")
		}
	case extractapi.JSONPathScheme:
		log.Info().Msg("extract is processing a jsonpath file")
		err = s.ExtractJSONPath(ctx)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in jsonpath processing ")
		}
	default:
		log.Error().Stack().Msg("invalid datasource scheme value " + schemeValue)
		os.Exit(1)
	}

	log.Info().Msg("schemeValue for rename is " + schemeValue)
	switch schemeValue {
	default:
		s.bumpMetric(churroDB)
		if schemeValue != extractapi.APIScheme && schemeValue != extractapi.HTTPPostScheme {
			s.renameFile(fileName)
		}
	}

	// dataprov_id, podname, poddate, recordsloaded

	return s
}

// Ping implements the Ping interface and simply responds by returning
// a response that also holds the current backpressure status
func (s *Server) Ping(ctx context.Context, size *pb.PingRequest) (hat *pb.PingResponse, err error) {
	return &pb.PingResponse{
		Backpressure: size.Backpressure,
	}, nil
}

func (s *Server) renameFile(path string) {
	newPath := path + ".churro-processed"
	err := os.Rename(path, newPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in renaming file")
	}
	log.Info().Msg("extract is renaming processed file " + newPath)
}

func (s *Server) bumpMetric(churroDB db.ChurroDatabase) {
	log.Info().Msg("extract bumping file processed metric")
	metrics, err := churroDB.GetExtractSourceMetrics(s.ExtractSource.ID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in getting extract source metrics")
		return
	}

	log.Info().Msg(fmt.Sprintf("found %d extractsource metric\n", len(metrics)))
	log.Info().Msg(fmt.Sprintf("extractsource metrics %+v\n", metrics))
	for i := 0; i < len(metrics); i++ {
		if metrics[i].Name == domain.MetricFilesProcessed {
			v, err := strconv.Atoi(metrics[i].Value)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in converting metric value")
				return
			}
			v++
			metrics[i].Value = strconv.Itoa(v)
			err = churroDB.UpdateExtractSourceMetric(metrics[i])
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in updating metric value")
				return
			}
		} else if metrics[i].Name == domain.MetricLastFileProcessed {
			metrics[i].Value = os.Getenv("CHURRO_FILENAME")
			err = churroDB.UpdateExtractSourceMetric(metrics[i])
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in updating metric value")
				return
			}
		}
	}

	pipelineMetrics, err := churroDB.GetAllPipelineMetrics()
	for i := 0; i < len(pipelineMetrics); i++ {
		if pipelineMetrics[i].Name == domain.MetricFilesProcessed {
			v, err := strconv.Atoi(pipelineMetrics[i].Value)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in converting pipeline metric value")
				return
			}
			v++
			pipelineMetrics[i].Value = strconv.Itoa(v)
			err = churroDB.UpdatePipelineMetric(pipelineMetrics[i])
			if err != nil {
				log.Error().Err(err).Msg("error in updating pipeline metric value")
				return
			}
		}
	}

}

func getColumns(wdir domain.ExtractSource) []extractapi.Column {

	cols := make([]extractapi.Column, 0)

	rules := wdir.ExtractRules
	for _, r := range rules {
		c := extractapi.Column{
			Name: r.ColumnName,
			Path: r.ColumnPath,
			Type: r.ColumnType,
		}
		cols = append(cols, c)
	}
	return cols
}

func getColumnNames(cols []extractapi.Column) (names []string) {
	for i := 0; i < len(cols); i++ {
		names = append(names, cols[i].Name)
	}
	return names
}

func getColumnTypes(cols []extractapi.Column) (types []string) {
	for i := 0; i < len(cols); i++ {
		types = append(types, cols[i].Type)
	}
	return types
}
