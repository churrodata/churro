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
	"net/http"
	"os"
	"time"

	"github.com/ohler55/ojg/oj"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/dataprov"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/transform"
)

// ExtractAPI Extract from an API that produces json messages...forever!
func (s *Server) ExtractAPI(ctx context.Context) (err error) {

	log.Info().Msg("ExtractAPI ...api URL " + s.ExtractSource.Path)

	// register to dataprov
	dp := domain.DataProvenance{
		Name: s.FileName,
		Path: s.FileName,
	}
	err = dataprov.Register(&dp, s.Pi, s.DBCreds)
	if err != nil {
		log.Error().Stack().Err(err).Msg("can not register data prov")
		os.Exit(1)
	}
	log.Info().Msg(fmt.Sprintf("dp info %+v", dp))

	// initialize with the single message default
	jsonStruct := extractapi.RawFormat{
		ColumnNames: []string{"metadata"},
		ColumnTypes: []string{"jsonb"},
	}

	// update if we have extract rules defined
	if len(s.ExtractSource.ExtractRules) > 0 {
		//jsonStruct.Columns = getColumns(s.ExtractSource)
		//jsonStruct.ColumnNames = getColumnNames(jsonStruct.Columns)
		//jsonStruct.ColumnTypes = getColumnTypes(jsonStruct.Columns)
	}

	log.Info().Msg(fmt.Sprintf("jsonStruct at top is %+v", jsonStruct))
	jsonStruct.Path = dp.Path
	jsonStruct.Dataprov = dp.ID

	apiurl := s.ExtractSource.Path

	jobProfile := domain.JobProfile{
		ID:               os.Getenv("CHURRO_EXTRACTLOG"),
		JobName:          os.Getenv("POD_NAME"),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		DataProvenanceID: dp.ID,
		FileName:         s.FileName,
		TableName:        s.ExtractSource.Tablename,
		RecordsLoaded:    0,
	}

	log.Info().Msg("setting table name to " + s.ExtractSource.Tablename)
	err = s.insertJobProfile(jobProfile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error inserting initial JobProfile")
		return
	}
	log.Info().Msg("inserted Extractlog")
	c := cron.New()
	cronExpression := "@every 30s"
	if s.ExtractSource.Cronexpression != "" {
		cronExpression = s.ExtractSource.Cronexpression
	}
	c.AddFunc(cronExpression, func() {
		// get a message from the API
		apiMessage, err := getMessage(apiurl)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error connecting to the database")
			return
		}

		var churroDB db.ChurroDatabase
		churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error creating the database")
			return
		}
		err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error connecting to the database")
			return
		}

		jsonStruct.Message = apiMessage
		jsonStruct.Records = make([]extractapi.GenericRow, 0)

		// see if user wants to extract certain stuff
		if len(s.ExtractSource.ExtractRules) > 0 {

			jsonStruct.Columns = make([]extractapi.Column, 0)
			jsonStruct.ColumnNames = make([]string, 0)
			jsonStruct.ColumnTypes = make([]string, 0)

			allCols := make([][]interface{}, 0)
			var rows int
			log.Info().Msg(fmt.Sprintf("user has %d extract rules defined\n", len(s.ExtractSource.ExtractRules)))
			for _, r := range s.ExtractSource.ExtractRules {

				ec := extractapi.Column{
					Name: r.ColumnName,
					Path: r.ColumnPath,
					Type: r.ColumnType,
				}
				jsonStruct.Columns = append(jsonStruct.Columns, ec)
				jsonStruct.ColumnNames = append(jsonStruct.ColumnNames, r.ColumnName)
				jsonStruct.ColumnTypes = append(jsonStruct.ColumnTypes, r.ColumnType)
				obj, parseError := oj.ParseString(string(apiMessage))
				if parseError != nil {
					log.Error().Stack().Err(parseError).Msg("error in oj.ParseString ")
					continue
				}

				cols, err := getJSONPathColumns(obj, r.ColumnPath)
				if err != nil {
					log.Error().Stack().Err(err)
					continue
				}
				allCols = append(allCols, cols)
				rows = len(cols)
			}

			for row := 0; row < rows; row++ {
				r := extractapi.GenericRow{
					Key: time.Now().UnixNano(),
				}
				r.Cols = make([]interface{}, 0)
				for cell := 0; cell < len(allCols); cell++ {
					r.Cols = append(r.Cols, allCols[cell][row])
				}

				// here is where we would transform...the r
				log.Info().Msg(fmt.Sprintf("cols going into transform %v\n", r.Cols))
				err := transform.RunRules(jsonStruct.ColumnNames, r.Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
				if err != nil {
					log.Error().Stack().Err(err)
				}
				log.Info().Msg(fmt.Sprintf("cols from transform %v\n", r.Cols))
				//

				jsonStruct.Records = append(jsonStruct.Records, r)
			}

		}

		err = s.tableCheck(jsonStruct.ColumnNames, jsonStruct.ColumnTypes)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error during tableCheck ")
			return
		}

		log.Info().Msg(fmt.Sprintf("jsonStruct has %d Records\n", len(jsonStruct.Records)))

		someBytes, _ := json.Marshal(jsonStruct)

		// write the message
		msg := extractapi.LoaderMessage{
			Key:        time.Now().UnixNano(),
			Metadata:   someBytes,
			DataFormat: extractapi.APIScheme,
		}

		s.process(jobProfile, churroDB, s.Pi.Spec.DataSource.Database, msg)

	})

	c.Start()

	for {
		log.Info().Msg("polling the API")
		if s.APIStopTime > 0 {
			log.Info().Msg(fmt.Sprintf("stopping the API, using APIStopTime of %d", s.APIStopTime))
			time.Sleep(time.Second * time.Duration(s.APIStopTime))
			c.Stop()
			return nil
		}
		time.Sleep(30 * time.Second)
	}

	return nil
}

func getMessage(url string) (msg []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return msg, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return msg, err
		}

		log.Info().Msg(fmt.Sprintf("read %d ", len(body)))

		//log.Info().Msg(string(body))
		return body, nil
	}
	return msg, fmt.Errorf("Invalid status from GET %+v", resp.StatusCode)
}

func (s *Server) insertJobProfile(jp domain.JobProfile) (err error) {
	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating the database")
		return err
	}
	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		return err
	}

	err = churroDB.CreateExtractLog(jp)

	return err
}
