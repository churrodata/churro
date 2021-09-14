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
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/dataprov"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/transform"
)

// ExtractXLS Excel file contents and exit
func (s *Server) ExtractXLS(ctx context.Context) (err error) {

	log.Info().Msg("ExtractXLS starting...")

	xlsxFile, err := excelize.OpenFile(s.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not open xlsx file" + s.FileName)
		return err
	}

	// TODO make sheetName configurable
	sheetName := "Sheet1"

	dp := domain.DataProvenance{
		Name: s.FileName,
		Path: s.FileName,
	}
	err = dataprov.Register(&dp, s.Pi, s.DBCreds)
	if err != nil {
		log.Error().Stack().Err(err).Msg("can not register data prov")
		os.Exit(1)
	}
	log.Info().Msg("dp info name " + dp.Name + " path " + dp.Path)

	var rows [][]string
	rows, err = xlsxFile.GetRows(sheetName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not GetRows xlsx file sheet " + sheetName)
		return err
	}

	xlsStruct := extractapi.XLSFormat{
		Path:         s.FileName,
		Dataprov:     dp.ID,
		PipelineName: s.Pi.Name,
		Columns:      getColumns(s.ExtractSource),
	}
	xlsStruct.ColumnNames = getColumnNames(xlsStruct.Columns)
	xlsStruct.ColumnTypes = getColumnTypes(xlsStruct.Columns)

	log.Info().Msg(fmt.Sprintf("wdir %+v", s.ExtractSource))
	log.Info().Msg(fmt.Sprintf("columns %+v", xlsStruct.Columns))
	log.Info().Msg(fmt.Sprintf("columnNames %+v", xlsStruct.ColumnNames))
	log.Info().Msg(fmt.Sprintf("columnTypes %+v", xlsStruct.ColumnTypes))
	log.Info().Msg(fmt.Sprintf("skipheaders %d", s.ExtractSource.Skipheaders))

	//firstRow := true
	// for now, only a single row per Queue message
	xlsStruct.Records = make([]extractapi.GenericRow, 0)

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating the database")
		os.Exit(1)
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		os.Exit(1)
	}

	jobProfile := domain.JobProfile{
		ID:               os.Getenv("CHURRO_EXTRACTLOG"),
		JobName:          os.Getenv("POD_NAME"),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		DataProvenanceID: dp.ID,
		FileName:         s.FileName,
		RecordsLoaded:    0,
	}

	// initialize the table to hold the xls data
	err = s.tableCheck(xlsStruct.ColumnNames, xlsStruct.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error doing tableCheck")
		os.Exit(1)
	}

	xlsStruct.Tablename = s.TableName

	var recordsRead int

	for r := 0; r < len(rows); r++ {

		record := rows[r]
		recordsRead++

		// process the xls header which we expect to be there
		if recordsRead <= s.ExtractSource.Skipheaders {
			log.Info().Msg(fmt.Sprintf("skipping header %d", recordsRead))
			/**

			firstRow = false
			err = s.tableCheck(xlsStruct.ColumnNames, xlsStruct.ColumnTypes)
			if err != nil {
				return err
			}
			xlsStruct.Tablename = s.TableName
			*/
		} else {

			r := getXLSRow(record, xlsStruct.Columns)

			// TODO apply transforms to XLS data
			err := transform.RunRules(xlsStruct.ColumnNames, r.Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in RunRules")
			}

			xlsStruct.Records = append(xlsStruct.Records, r)
			log.Debug().Msg("xls record read")

			log.Info().Msg("pushing to Queue")
			xlsBytes, _ := json.Marshal(xlsStruct)
			msg := extractapi.LoaderMessage{
				Metadata:   xlsBytes,
				DataFormat: extractapi.XLSXScheme,
			}

			s.process(jobProfile, churroDB, s.Pi.Spec.DataSource.Database, msg)
			xlsStruct.Records = make([]extractapi.GenericRow, 0)
		}

	}

	jobProfile.RecordsLoaded = len(rows)

	err = churroDB.CreateExtractLog(jobProfile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in createextractlog")
	}

	log.Info().Msg("end of xlsx file reached, cancelling pushes...")

	return err
}

func getXLSRow(record []string, cols []extractapi.Column) extractapi.GenericRow {
	xlsRow := extractapi.GenericRow{
		Key:  time.Now().UnixNano(),
		Cols: make([]interface{}, len(cols)),
	}

	for i := 0; i < len(cols); i++ {
		path, err := strconv.Atoi(cols[i].Path)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in converting col path to int")
			continue
		}
		xlsRow.Cols[i] = record[path]
	}

	return xlsRow
}

func genColumnNames(colCount int) []string {
	cols := make([]string, 0)
	charsetCount := 24
	startingChar := 'A'
	currentChar := startingChar
	prefixChar := startingChar
	currentprefix := ""
	charsDone := 0

	for i := 0; i < colCount; i++ {
		if charsDone >= charsetCount {
			currentprefix = string(prefixChar)
			prefixChar = prefixChar + 1
			startingChar = startingChar
			charsDone = 0
			currentChar = rune(int(startingChar))
		}
		asInt := int(currentChar)
		cols = append(cols, currentprefix+fmt.Sprintf("%d", asInt))
		currentChar = rune(int(currentChar) + 1)
		charsDone++
	}
	return cols
}
