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
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"time"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/dataprov"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/transform"
	"github.com/rs/zerolog/log"
	"gopkg.in/xmlpath.v2"
)

type compiledXMLRule struct {
	rule         domain.ExtractRule
	compiledRule *xmlpath.Path
}

// ExtractXML file contents and exit
func (s *Server) ExtractXML(ctx context.Context) (err error) {

	log.Info().Msg("ExtractXML starting...")

	// read the XML file to be processed and parse it
	reader, err := os.Open(s.FileName)
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = func(label string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	root, err := xmlpath.ParseDecoder(decoder)
	if err != nil {
		return err
	}

	// register data provenance
	dp := domain.DataProvenance{
		Name: s.FileName,
		Path: s.FileName,
	}
	err = dataprov.Register(&dp, s.Pi, s.DBCreds)
	if err != nil {
		log.Error().Stack().Err(err).Msg("can not register data prov")
		os.Exit(1)
	}
	log.Info().Msg("dp info " + dp.Name + dp.Path)

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating the database")
		os.Exit(1)
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.AdminDataSource)

	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		os.Exit(1)
	}

	rules := getXMLRules(s.ExtractSource)

	xmlStruct := getXMLFormat(rules, root)

	xmlStruct.Path = s.FileName
	xmlStruct.Dataprov = dp.ID
	xmlStruct.PipelineName = s.Pi.Name
	xmlStruct.Tablename = s.TableName

	recLen := len(xmlStruct.Records)
	log.Info().Msg(fmt.Sprintf("xml records to process %d\n", recLen))

	err = s.tableCheck(xmlStruct.ColumnNames, xmlStruct.ColumnTypes)
	if err != nil {
		return err
	}

	// partStruct holds a portion of the overall xmlStruct records
	// since we push portions of the overall set of records
	partStruct := xmlStruct
	partStruct.Records = make([]extractapi.GenericRow, 0)

	log.Info().Msg(fmt.Sprintf("partStruct col len %d\n", len(partStruct.ColumnNames)))

	recordsProcessed := 0
	for i := 0; i < recLen; i++ {
		log.Info().Msg(fmt.Sprintf("before transform %v", xmlStruct.Records[i].Cols))
		err := transform.RunRules(xmlStruct.ColumnNames, xmlStruct.Records[i].Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in RunRules ")
		}
		log.Info().Msg(fmt.Sprintf("after transform %+v", xmlStruct.Records[i].Cols))

		recordsProcessed++
		xmlStruct.Records[i].Key = time.Now().UnixNano()

		partStruct.Records = append(partStruct.Records, xmlStruct.Records[i])
		log.Info().Msg("pushing to Queue")
		//convert partStruct into []byte
		xmlBytes, _ := json.Marshal(partStruct)
		msg := extractapi.LoaderMessage{}
		msg.Metadata = xmlBytes
		msg.DataFormat = "xml"
		jobProfile := domain.JobProfile{
			ID:               os.Getenv("CHURRO_EXTRACT"),
			JobName:          os.Getenv("POD_NAME"),
			StartDate:        time.Now().Format("2006-01-02 15:04:05"),
			DataProvenanceID: dp.ID,
			FileName:         s.FileName,
			RecordsLoaded:    recordsProcessed,
		}

		err = churroDB.CreateExtractLog(jobProfile)
		if err != nil {
			log.Error().Stack().Err(err)
		}

		s.process(jobProfile, churroDB, s.Pi.Spec.DataSource.Database, msg)
		recordsProcessed = 0
		partStruct.Records = make([]extractapi.GenericRow, 0)

	}

	log.Info().Msg("end of XML file reached, cancelling pushes...")

	return err
}

func getColumn(rule compiledXMLRule, root *xmlpath.Node) (cols []string) {
	iter := rule.compiledRule.Iter(root)
	for iter.Next() {
		cols = append(cols, iter.Node().String())
	}
	return cols
}

func getXMLRules(extractSource domain.ExtractSource) (rules []compiledXMLRule) {
	if extractSource.Scheme == extractapi.XMLScheme {
		for _, v := range extractSource.ExtractRules {
			fromRule := v
			path, err := xmlpath.Compile(fromRule.ColumnPath)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in compiling xmlpath")
				break
			}
			rules = append(rules, compiledXMLRule{rule: fromRule, compiledRule: path})
		}
	}
	return rules
}

func getXMLFormat(rules []compiledXMLRule, root *xmlpath.Node) (format extractapi.GenericFormat) {

	cols := make([][]string, 0)

	for i := 0; i < len(rules); i++ {
		format.ColumnNames = append(format.ColumnNames, rules[i].rule.ColumnName)
		format.ColumnTypes = append(format.ColumnTypes, "TEXT")

		cols = append(cols, getColumn(rules[i], root))
	}

	var records int
	log.Info().Msg(fmt.Sprintf("cols %+v\n", cols))
	if len(cols) > 0 {
		records = len(cols[0])
	} else {
		log.Info().Msg("no columns found in queries")
		return format
	}
	columns := len(cols)
	for rec := 0; rec < records; rec++ {
		xmlrow := extractapi.GenericRow{
			Key: time.Now().UnixNano(),
		}
		for c := 0; c < columns; c++ {
			xmlrow.Cols = append(xmlrow.Cols, cols[c][rec])
		}
		format.Records = append(format.Records, xmlrow)
	}

	return format
}
