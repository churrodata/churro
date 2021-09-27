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
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
)

// ExtractJSON Extract a JSON file contents and exit
// this file is expected to be a single JSON document
func (s *Server) ExtractJSON(ctx context.Context) (err error) {

	log.Info().Msg("ExtractJSON starting...\n")

	jsonfile, err := os.Open(s.FileName)
	if err != nil {
		return fmt.Errorf("could not open JSON file: %s %v", s.FileName, err)
	}
	defer jsonfile.Close()

	colNames := []string{"metadata"}
	colTypes := []string{"jsonb"}

	err = s.tableCheck(colNames, colTypes)
	if err != nil {
		return err
	}

	var byteValue []byte
	byteValue, err = ioutil.ReadAll(jsonfile)
	if err != nil {
		return fmt.Errorf("can not read json input file %v", err)
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(byteValue), &result)
	if err != nil {
		return fmt.Errorf("can not unmarshal json input file %v", err)
	}

	jsonStruct := extractapi.IntermediateFormat{
		Path:        s.DP.Path,
		Dataprov:    s.DP.ID,
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Messages:    make([]map[string]interface{}, 0),
	}
	jsonStruct.Messages = append(jsonStruct.Messages, result)

	someBytes, _ := json.Marshal(jsonStruct)

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		os.Exit(1)
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		os.Exit(1)
	}

	msg := extractapi.LoaderMessage{
		Key:        time.Now().UnixNano(),
		Metadata:   someBytes,
		DataFormat: extractapi.JSONScheme,
	}

	jobProfile := domain.JobProfile{
		ID:               os.Getenv("CHURRO_EXTRACTLOG"),
		JobName:          os.Getenv("POD_NAME"),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		DataProvenanceID: s.DP.ID,
		FileName:         s.FileName,
		RecordsLoaded:    1,
	}

	err = churroDB.CreateExtractLog(jobProfile)
	if err != nil {
		log.Error().Stack().Err(err)
	}

	s.process(jobProfile, churroDB, s.Pi.Spec.DataSource.Database, msg)

	return err
}
