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
	"encoding/json"
	"fmt"
	"net/http"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/stats"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func (s *Server) process(jp domain.JobProfile, xyz db.ChurroDatabase, database string, elem extractapi.LoaderMessage) {
	switch s.SchemeValue {
	case extractapi.FinnHubScheme:
		s.processFinnhubStocks(jp, xyz, database, elem)
	case extractapi.APIScheme:
		s.processAPI(jp, xyz, database, elem)
	case extractapi.XMLScheme:
		s.processXML(jp, xyz, database, elem)
	case extractapi.XLSXScheme:
		s.processXLSX(jp, xyz, database, elem)
	case extractapi.CSVScheme:
		s.processCSV(jp, xyz, database, elem)
	case extractapi.JSONPathScheme:
		s.processJSONPath(jp, xyz, database, elem)
	case extractapi.JSONScheme:
		s.processJSON(jp, xyz, database, elem)
	default:
		log.Error().Stack().Msg("invalid scheme found in process " + s.SchemeValue)
	}

	s.processExtensions(elem)
}

func (s *Server) processCSV(jp domain.JobProfile, churroDB db.ChurroDatabase, database string, elem extractapi.LoaderMessage) {

	//unmarshal elem metadata into CSV message
	var csvMsg extractapi.CSVFormat
	err := json.Unmarshal(elem.Metadata, &csvMsg)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in unmarshal")
		return
	}

	err = churroDB.GetBulkInsertStatement(extractapi.CSVScheme, database, csvMsg.Tablename, csvMsg.ColumnNames, csvMsg.Records, csvMsg.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in bulk insert ")
		return
	}

	t := stats.PipelineStats{
		DataprovID: csvMsg.Dataprov,
		Pipeline:   csvMsg.PipelineName,
		FileName:   csvMsg.Path,
		RecordsIn:  int64(len(csvMsg.Records)),
	}

	err = churroDB.UpdatePipelineStats(t)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in stats update ")
		return
	}
}

func (s *Server) processJSON(jp domain.JobProfile, churroDB db.ChurroDatabase, pipelineName string, elem extractapi.LoaderMessage) {
	colNames := []string{"metadata"}
	cols := []interface{}{string(elem.Metadata)}
	err := churroDB.GetInsertStatement(extractapi.JSONScheme, pipelineName, s.TableName, colNames, cols, elem.Key)
	if err != nil {
		return
	}

}

func (s *Server) startMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in startMetrics")
	}
}

func (s *Server) processXML(jp domain.JobProfile, churroDB db.ChurroDatabase, database string, elem extractapi.LoaderMessage) {

	//unmarshal elem metadata into XML message
	var xmlMsg extractapi.XMLFormat
	err := json.Unmarshal(elem.Metadata, &xmlMsg)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in processXML")
		return
	}

	log.Info().Msg(fmt.Sprintf("loader is processing XML records %d", len(xmlMsg.Records)))
	log.Info().Msg(fmt.Sprintf("loader is processing XML columns %v", xmlMsg.ColumnNames))

	err = churroDB.GetBulkInsertStatement(extractapi.XMLScheme, database, xmlMsg.Tablename, xmlMsg.ColumnNames, xmlMsg.Records, xmlMsg.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on bulk insert")
		return
	}

	t := stats.PipelineStats{
		DataprovID: xmlMsg.Dataprov,
		Pipeline:   xmlMsg.PipelineName,
		FileName:   xmlMsg.Path,
		RecordsIn:  int64(len(xmlMsg.Records)),
	}

	err = churroDB.UpdatePipelineStats(t)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on stats update")
	}
}

func (s *Server) processFinnhubStocks(jp domain.JobProfile, churroDB db.ChurroDatabase, database string, elem extractapi.LoaderMessage) {

	//unmarshal elem metadata into CSV message
	var csvMsg extractapi.CSVFormat
	err := json.Unmarshal(elem.Metadata, &csvMsg)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on csv unmarshal")
		return
	}

	for _, r := range csvMsg.Records {
		err := churroDB.GetInsertStatement(extractapi.FinnHubScheme, database, csvMsg.Tablename, csvMsg.ColumnNames, r.Cols, r.Key)
		if err != nil {
			return
		}
	}

	t := stats.PipelineStats{
		DataprovID: csvMsg.Dataprov,
		Pipeline:   csvMsg.PipelineName,
		FileName:   csvMsg.Path,
		RecordsIn:  int64(len(csvMsg.Records)),
	}

	err = churroDB.UpdatePipelineStats(t)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in stats update")
	}
}

func (s *Server) processXLSX(jp domain.JobProfile, churroDB db.ChurroDatabase, database string, elem extractapi.LoaderMessage) {

	//unmarshal elem metadata into XLS message
	var xlsMsg extractapi.XLSFormat
	err := json.Unmarshal(elem.Metadata, &xlsMsg)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in xls unmarshal")
		return
	}

	err = churroDB.GetBulkInsertStatement(extractapi.XLSXScheme, database, xlsMsg.Tablename, xlsMsg.ColumnNames, xlsMsg.Records, xlsMsg.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in bulk insert")
		return
	}

	t := stats.PipelineStats{
		DataprovID: xlsMsg.Dataprov,
		Pipeline:   xlsMsg.PipelineName,
		FileName:   xlsMsg.Path,
		RecordsIn:  int64(len(xlsMsg.Records)),
	}

	err = churroDB.UpdatePipelineStats(t)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in stats update")
		return
	}
}

func (s *Server) processJSONPath(jp domain.JobProfile, churroDB db.ChurroDatabase, database string, elem extractapi.LoaderMessage) {

	//unmarshal into JsonPathMessage
	var jsonPathMsg extractapi.JsonPathFormat
	err := json.Unmarshal(elem.Metadata, &jsonPathMsg)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in jsonpath unmarshal")
		return
	}

	log.Info().Msg(fmt.Sprintf("jsonPathMsg %+v\n", jsonPathMsg))

	err = churroDB.GetBulkInsertStatement(extractapi.JSONPathScheme, database, jsonPathMsg.Tablename, jsonPathMsg.ColumnNames, jsonPathMsg.Records, jsonPathMsg.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in bulk insert")
		return
	}

	t := stats.PipelineStats{
		DataprovID: jsonPathMsg.Dataprov,
		Pipeline:   jsonPathMsg.PipelineName,
		FileName:   jsonPathMsg.Path,
		RecordsIn:  int64(len(jsonPathMsg.Records)),
	}

	err = churroDB.UpdatePipelineStats(t)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in jsonpath stats update")
		return
	}

}

func (s *Server) processAPI(jp domain.JobProfile, churroDB db.ChurroDatabase, pipelineName string, elem extractapi.LoaderMessage) {
	var jsonStruct extractapi.RawFormat
	err := json.Unmarshal(elem.Metadata, &jsonStruct)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in unmarshal ")
		return
	}

	cols := make([]interface{}, 1)
	cols[0] = string(jsonStruct.Message)

	t := stats.PipelineStats{
		DataprovID: jsonStruct.Dataprov,
		Pipeline:   s.Pi.Name,
		FileName:   s.ExtractSource.Path,
		RecordsIn:  int64(1),
	}

	recCount := len(jsonStruct.Records)
	if recCount > 0 {
		t.RecordsIn = int64(recCount)
		err := churroDB.GetBulkInsertStatement(extractapi.APIScheme, pipelineName, s.TableName, jsonStruct.ColumnNames, jsonStruct.Records, jsonStruct.ColumnTypes)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in bulk insert")
			return
		}
	} else {
		err := churroDB.GetInsertStatement(extractapi.APIScheme, pipelineName, s.TableName, jsonStruct.ColumnNames, cols, elem.Key)
		if err != nil {
			return
		}
	}

	err = churroDB.UpdatePipelineStats(t)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in stats update")
		return
	}

	jp.RecordsLoaded = recCount

	var jp2 domain.JobProfile
	jp2, err = churroDB.GetExtractLogById(jp.ID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in jobprofile update")
		return
	}
	jp2.RecordsLoaded += recCount

	err = churroDB.UpdateExtractLog(jp2)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in jobprofile update")
		return
	}

}
