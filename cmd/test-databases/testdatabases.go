// Copyright 2021 churrodata LLC
// Author: djm

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/stats"
	"github.com/churrodata/churro/pkg/config"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const testNamespace = "test"
const testChurroDatabase = "testchurro"

func main() {
	zerolog.TimeFieldFormat = time.RFC822

	log.Logger = log.With().Caller().Logger()

	log.Info().Msg("testdatabases")
	dbTypeFlag := flag.String("dbtype", "mysql", "either mysql or cockroachdb")

	flag.Parse()

	if *dbTypeFlag == "" {
		log.Error().Msg("dbtypeflag required")
		os.Exit(1)
	}
	log.Info().Msg("testdatabases dbtype=" + *dbTypeFlag)

	churroDB, err := db.NewChurroDB(*dbTypeFlag)
	if err != nil {
		log.Error().Err(err)
		os.Exit(1)
	}

	err = os.Setenv("CHURRO_NAMESPACE", testNamespace)

	var creds config.DBCredentials
	var source v1alpha1.Source
	if *dbTypeFlag == "mysql" {
		creds = config.DBCredentials{
			Username: "root",
			Password: "not-so-secure",
		}
		source = v1alpha1.Source{
			Host:     "127.0.0.1",
			Port:     3306,
			Database: "mysql",
			Username: "root",
			Password: "not-so-secure",
		}
	} else if *dbTypeFlag == "cockroachdb" {
		creds = config.DBCredentials{
			//Username: "root",
			//Password: "not-so-secure",
			CACertPath: "ca.crt",
			//CAKeyPath:       "/tmp",
			SSLRootCertPath: "client.root.crt",
			SSLRootKeyPath:  "client.root.key",
			SSLCertPath:     "node.crt",
			SSLKeyPath:      "node.key",
		}
		source = v1alpha1.Source{
			Host:     "127.0.0.1",
			Port:     26257,
			Database: "mydb",
			Username: "root",
			//Password: "not-so-secure",
		}
	} else {
		log.Error().Msg("invalid dbtype")
		os.Exit(1)
	}

	// as the pipeline user
	err = churroDB.GetConnection(creds, source)
	if err != nil {
		log.Error().Err(err)
		os.Exit(1)
	}

	version, err := churroDB.GetVersion()
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}
	log.Info().Msg(version)

	creds.Username = testNamespace
	err = churroDB.CreateUser(creds.Username, creds.Password)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	err = churroDB.CreatePipelineDatabase(testNamespace)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	err = churroDB.CreatePipelineObjects(testNamespace, creds.Username)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	job := domain.JobProfile{
		ID:               xid.New().String(),
		DataProvenanceID: xid.New().String(),
		JobName:          xid.New().String(),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		RecordsLoaded:    1,
	}

	source.Database = testNamespace
	err = churroDB.GetConnection(creds, source)
	if err != nil {
		log.Error().Err(err)
		os.Exit(1)
	}

	err = churroDB.CreateExtractLog(job)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	job.RecordsLoaded = 2
	err = churroDB.UpdateExtractLog(job)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	_, err = churroDB.GetExtractLogById(job.ID)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	stat := stats.PipelineStats{
		ID:         int64(time.Now().Unix()),
		DataprovID: "something",
		Pipeline:   testNamespace,
		FileName:   "somefile.csv",
		RecordsIn:  int64(3),
	}

	err = churroDB.UpdatePipelineStats(stat)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	// extract logic
	colTypes := []string{extractapi.COLTYPE_VARCHAR, extractapi.COLTYPE_VARCHAR}
	colNames := []string{"cola", "colb"}
	colValues := []interface{}{"value1", "value2"}
	tableName := "mycsvtable"

	err = churroDB.CreateTable(creds.Username, testNamespace, tableName, colNames, colTypes)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	initialized := churroDB.IsInitialized(tableName)
	if !initialized {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	key := int64(time.Now().Unix())
	scheme := "csv"
	err = churroDB.GetInsertStatement(scheme, testNamespace, tableName, colNames, colValues, key)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	records := make([]extractapi.GenericRow, 0)
	row1 := extractapi.GenericRow{
		Key:  key + 1,
		Cols: []interface{}{"one", "two"},
	}
	records = append(records, row1)
	row2 := extractapi.GenericRow{
		Key:  key + 2,
		Cols: []interface{}{"one", "two"},
	}
	records = append(records, row2)

	err = churroDB.GetBulkInsertStatement(scheme, testNamespace, tableName, colNames, records, colTypes)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	err = churroDB.CreateObjects(testChurroDatabase)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	// connect to the churro database

	source.Database = testChurroDatabase
	err = churroDB.GetConnection(creds, source)
	if err != nil {
		log.Error().Err(err)
		os.Exit(1)
	}

	// churro.extractsourcemetric
	eMetric := domain.ExtractSourceMetric{
		Value:           "foo",
		ExtractSourceID: xid.New().String(),
		Name:            "flk",
	}

	err = churroDB.CreateExtractSourceMetric(eMetric)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	err = churroDB.UpdateExtractSourceMetric(eMetric)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	mets, err := churroDB.GetExtractSourceMetrics(eMetric.ExtractSourceID)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}
	if len(mets) == 0 {
		log.Error().Msg("error in GetExtractSourceMetrics")
		os.Exit(1)
	}

	// churro.pipelinemetrics
	metric := domain.PipelineMetric{
		Name:  xid.New().String(),
		Value: "myvalue",
	}
	err = churroDB.CreatePipelineMetric(metric)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	err = churroDB.UpdatePipelineMetric(metric)
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}

	metrics, err := churroDB.GetAllPipelineMetrics()
	if err != nil {
		log.Error().Err(err).Msg("some error")
		os.Exit(1)
	}
	log.Info().Msg(fmt.Sprintf("metrics %d", len(metrics)))

}
