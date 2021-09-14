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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/dataprov"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/transform"

	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
)

// ExtractJSONPath Extract a JSON file contents using jsonpath rules and exit
// this file is expected to be a single JSON document
func (s *Server) ExtractJSONPath(ctx context.Context) (err error) {

	log.Debug().Msg("ExtractJSONPath starting...")

	byteValue, err := ioutil.ReadFile(s.FileName)
	if err != nil {
		return fmt.Errorf("could not open JSONPath file: %s %v", s.FileName, err)
	}

	obj, parseError := oj.ParseString(string(byteValue))
	if parseError != nil {
		return fmt.Errorf("error parsing rule: %s %v", string(byteValue), err)
	}
	dp := domain.DataProvenance{
		Name: s.FileName,
		Path: s.FileName,
	}
	err = dataprov.Register(&dp, s.Pi, s.DBCreds)
	if err != nil {
		return fmt.Errorf("can not register data prov %v %v", dp, err)
	}
	log.Debug().Msg(fmt.Sprintf("dp info %v", dp))

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return err
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		return err
	}

	jsonStruct := extractapi.JsonPathFormat{
		Path:         dp.Path,
		Dataprov:     dp.ID,
		PipelineName: s.Pi.Name,
	}
	log.Info().Msg(fmt.Sprintf("wdir %+v\n", s.ExtractSource))

	// since we assume a single json message, we extract that
	// into a single record with multiple columns
	jsonStruct.Records = make([]extractapi.GenericRow, 0)

	allCols := make([][]interface{}, 0)

	jsonStruct.Tablename = s.TableName

	var rows int

	jsonStruct.Columns = make([]extractapi.Column, 0)
	jsonStruct.ColumnNames = make([]string, 0)
	jsonStruct.ColumnTypes = make([]string, 0)

	for _, r := range s.ExtractSource.ExtractRules {
		ec := extractapi.Column{
			Name: r.ColumnName,
			Path: r.ColumnPath,
			Type: r.ColumnType,
		}

		jsonStruct.Columns = append(jsonStruct.Columns, ec)
		jsonStruct.ColumnNames = append(jsonStruct.ColumnNames, r.ColumnName)
		jsonStruct.ColumnTypes = append(jsonStruct.ColumnTypes, r.ColumnType)
		cols, err := getJSONPathColumns(obj, r.ColumnPath)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}
		log.Info().Msg(fmt.Sprintf("jeff cols here is %+v", cols))
		allCols = append(allCols, cols)
		rows = len(cols)
	}

	log.Info().Msg(fmt.Sprintf("columns %+v", jsonStruct.Columns))
	log.Info().Msg(fmt.Sprintf("columnNames %+v", jsonStruct.ColumnNames))
	log.Info().Msg(fmt.Sprintf("columnTypes %+v", jsonStruct.ColumnTypes))

	err = s.tableCheck(jsonStruct.ColumnNames, jsonStruct.ColumnTypes)
	if err != nil {
		return err
	}

	for row := 0; row < rows; row++ {
		r := extractapi.GenericRow{
			Key: time.Now().UnixNano(),
		}
		r.Cols = make([]interface{}, 0)
		for cell := 0; cell < len(allCols); cell++ {
			r.Cols = append(r.Cols, allCols[cell][row])
		}

		err := transform.RunRules(jsonStruct.ColumnNames, r.Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in run rules")
		}
		jsonStruct.Records = append(jsonStruct.Records, r)
	}

	someBytes, _ := json.Marshal(jsonStruct)

	msg := extractapi.LoaderMessage{Metadata: someBytes, DataFormat: extractapi.JSONPathScheme}
	jobProfile := domain.JobProfile{
		ID:               os.Getenv("CHURRO_EXTRACTLOG"),
		JobName:          os.Getenv("POD_NAME"),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		DataProvenanceID: dp.ID,
		FileName:         s.FileName,
		RecordsLoaded:    rows,
	}

	err = churroDB.CreateExtractLog(jobProfile)
	if err != nil {
		log.Error().Stack().Err(err)
	}

	s.process(jobProfile, churroDB, s.Pi.Spec.DataSource.Database, msg)

	log.Info().Msg("end of jsonpath file reached, cancelling pushes...")

	return err
}

func getJSONPathColumns(parsedFileBytes interface{}, jsonpath string) (columns []interface{}, err error) {
	var x jp.Expr
	log.Info().Msg("getJSONPathColumns " + jsonpath)
	x, err = jp.ParseString(jsonpath)
	if err != nil {
		return columns, err
	}
	var cols []interface{}
	cols = x.Get(parsedFileBytes)
	log.Info().Msg(fmt.Sprintf("cols here %d", len(cols)))

	for i := 0; i < len(cols); i++ {
		var s interface{}
		if cols[i] == nil {
			s = "null"
		} else {
			s = cols[i]
		}
		columns = append(columns, s)
	}
	return columns, nil
}
