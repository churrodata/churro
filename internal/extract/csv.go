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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/transform"
)

// ExtractCSV Extract a CSV file contents and exit
func (s *Server) ExtractCSV(ctx context.Context) (err error) {

	log.Info().Msg("ExtractCSV starting...")

	csvfile, err := os.Open(s.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not open csv file " + s.FileName)
		return err
	}

	r := csv.NewReader(csvfile)

	csvStruct := extractapi.GenericFormat{
		Path:         s.FileName,
		Dataprov:     s.DP.ID,
		PipelineName: s.Pi.Name,
		Columns:      sortByPath(getColumns(s.ExtractSource)),
	}
	csvStruct.ColumnNames = getColumnNames(csvStruct.Columns)
	csvStruct.ColumnTypes = getColumnTypes(csvStruct.Columns)
	log.Info().Msg(fmt.Sprintf("wdir %+v", s.ExtractSource))
	log.Info().Msg(fmt.Sprintf("columns %+v", csvStruct.Columns))
	log.Info().Msg(fmt.Sprintf("columnNames %+v", csvStruct.ColumnNames))
	log.Info().Msg(fmt.Sprintf("columnTypes %+v", csvStruct.ColumnTypes))
	log.Info().Msg(fmt.Sprintf("skipheaders %d", s.ExtractSource.Skipheaders))

	//firstRow := true
	csvStruct.Records = make([]extractapi.GenericRow, 0)

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

	// initialize the table that will hold the csv data
	err = s.tableCheck(csvStruct.ColumnNames, csvStruct.ColumnTypes)
	if err != nil {
		return err
	}
	csvStruct.Tablename = s.TableName

	var recordsRead int

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		recordsRead++

		// process the csv header which we expect to be there
		if recordsRead <= s.ExtractSource.Skipheaders {
			log.Info().Msg(fmt.Sprintf("skipping header %d", recordsRead))
		} else {
			tmp := make([]interface{}, len(record))
			for i, v := range record {
				tmp[i] = v
			}

			r := getCSVRow(record, csvStruct.Columns)
			log.Info().Msg(fmt.Sprintf("row from GetCSVRow %v\n", r))
			err := transform.RunRules(csvStruct.ColumnNames, r.Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in runRules")
			}

			csvStruct.Records = append(csvStruct.Records, r)
			log.Debug().Msg(fmt.Sprintf("csv record read %v", record))

			log.Info().Msg("pushing to Queue")
			//convert csvStruct into []byte
			csvBytes, _ := json.Marshal(csvStruct)
			msg := extractapi.LoaderMessage{
				Metadata:   csvBytes,
				DataFormat: extractapi.CSVScheme,
			}

			jobProfile := domain.JobProfile{
				ID:               os.Getenv("CHURRO_EXTRACTLOG"),
				JobName:          os.Getenv("POD_NAME"),
				StartDate:        time.Now().Format("2006-01-02 15:04:05"),
				DataProvenanceID: s.DP.ID,
				FileName:         s.FileName,
				TableName:        s.TableName,
				RecordsLoaded:    recordsRead,
			}
			log.Info().Msg(fmt.Sprintf("creating extract log of %v", jobProfile))

			err = churroDB.CreateExtractLog(jobProfile)
			if err != nil {
				log.Error().Stack().Err(err)
			}

			s.process(jobProfile, churroDB, s.Pi.Spec.DataSource.Database, msg)
			csvStruct.Records = make([]extractapi.GenericRow, 0)
		}

	}

	log.Info().Msg("end of CSV file reached, cancelling pushes...")

	// TODO , here is where we insert a row into the extractlog table
	// dataprov_id, podname, poddate, recordsloaded
	// dp.Id, env var POD_NAME, (current time)poddate, recordsloaded

	return err
}

func getCSVRow(record []string, cols []extractapi.Column) extractapi.GenericRow {
	csvRow := extractapi.GenericRow{
		Key:  time.Now().UnixNano(),
		Cols: make([]interface{}, len(cols)),
	}
	for i := 0; i < len(cols); i++ {
		path, err := strconv.Atoi(cols[i].Path)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in converting col path to int")
			continue
		}
		csvRow.Cols[i] = record[path]
	}

	return csvRow
}

func sortByPath(inCols []extractapi.Column) []extractapi.Column {

	sort.Slice(inCols, func(i, j int) bool {
		return inCols[i].Path < inCols[j].Path
	})

	return inCols
}
