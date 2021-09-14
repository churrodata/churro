// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package cockroachdb

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/stats"
	"github.com/churrodata/churro/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	DB_COCKROACH = "cockroachdb"
)

type CockroachChurroDatabase struct {
	Connection *sql.DB
	namespace  string
}

func (d CockroachChurroDatabase) GetVersion() (string, error) {
	sqlStr := fmt.Sprintf("SELECT VERSION()")
	log.Info().Msg(sqlStr)
	stmt, err := d.Connection.Prepare(sqlStr)
	if err != nil {
		return "", err
	}
	var version string
	err = stmt.QueryRow().Scan(&version)
	if err != nil {
		return "", err
	}
	return version, nil
}

func (d CockroachChurroDatabase) CreateObjects(dbName string) error {

	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database" + dbName)

	/**
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.watchdirectory ( id STRING PRIMARY KEY, name STRING NOT NULL, path STRING NOT NULL UNIQUE, scheme STRING NOT NULL, regex STRING NOT NULL, cronexpression STRING, tablename STRING NOT NULL, lastupdated TIMESTAMP);", dbName)
	log.Info().Msg(sqlStr)
	var stmt *sql.Stmt
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg("watchdirectory Table created successfully..")
	*/
	/**
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractrule ( id STRING PRIMARY KEY, watchdirectoryid STRING references %s.watchdirectory(id) on update cascade on delete cascade, columnname STRING NOT NULL, columnpath STRING NOT NULL, columntype STRING NOT NULL, matchvalues STRING, transformfunction string, lastupdated TIMESTAMP);", dbName, dbName)
	log.Info().Msg(sqlStr)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg("extractrule Table created successfully..")
	*/
	/**
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extension ( id STRING PRIMARY KEY, watchdirectoryid STRING references %s.watchdirectory(id) on update cascade on delete cascade, extensionname STRING NOT NULL, extensionpath STRING NOT NULL, lastupdated TIMESTAMP);", dbName, dbName)
	log.Info().Msg(sqlStr)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg("extension Table created successfully..")
	*/
	/**
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.transformfunction ( id STRING PRIMARY KEY, name STRING NOT NULL, source STRING NOT NULL, lastupdated TIMESTAMP);", dbName)
	log.Info().Msg(sqlStr)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg("transformfunction Table created successfully..")
	*/
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.pipelinemetric ( name STRING PRIMARY KEY, value STRING NOT NULL, lastupdated TIMESTAMP);", dbName)
	log.Info().Msg(sqlStr)
	stmt, err := d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg("pipelinemetric Table created successfully..")
	// select to see if we have already seeded the metric table
	sqlStr = fmt.Sprintf("SELECT name from %s.pipelinemetric where name = '%s'", dbName, domain.MetricFilesProcessed)
	log.Info().Msg(sqlStr)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}

	var xname string
	err = stmt.QueryRow().Scan(&xname)
	if err == sql.ErrNoRows {
		sqlStr = fmt.Sprintf("INSERT into %s.pipelinemetric ( name, value, lastupdated) values ('%s', '0', now());", dbName, domain.MetricFilesProcessed)
		log.Info().Msg(sqlStr)
		stmt, err = d.Connection.Prepare(sqlStr)
		if err != nil {
			return err
		}
		_, err = stmt.Exec()
		if err != nil {
			return err
		}
		log.Info().Msg("pipelinemetric Table insert success..")
	} else {
		log.Info().Msg("pipelinemetric Table already seeded..")
	}
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractsourcemetric ( extractsourceid STRING, name STRING NOT NULL, value STRING NOT NULL, lastupdated TIMESTAMP);", dbName)
	log.Info().Msg(sqlStr)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg("extractsourcemetric Table created successfully..")

	return nil
}
func (d CockroachChurroDatabase) GetDatabaseType() string {
	return DB_COCKROACH
}

func (d *CockroachChurroDatabase) GetConnection(dbCreds config.DBCredentials, source v1alpha1.Source) (err error) {
	d.namespace = os.Getenv("CHURRO_NAMESPACE")
	if d.namespace == "" {
		log.Error().Stack().Msg("error CHURRO_NAMESPACE is empty")
		return fmt.Errorf("CHURRO_NAMESPACE env var required")
	}

	pgConnectString := dbCreds.GetDBConnectString(source)
	log.Info().Msg(pgConnectString)
	d.Connection, err = sql.Open("postgres", pgConnectString)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in GetConnection")
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) CreateDataprov(data domain.DataProvenance) error {
	insertStmt, err := d.Connection.Prepare("INSERT into DATAPROV (id, name, path, lastupdated) values ($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	if _, err := insertStmt.Exec(data.ID, data.Name, data.Path, data.LastUpdated); err != nil {
		return err
	}
	return nil
}

func (d *CockroachChurroDatabase) UpdatePipelineStats(data stats.PipelineStats) error {
	var recordsIn int64
	// get existing records count if a row exists
	sqlstr := fmt.Sprintf("SELECT records_in from %s.pipeline_stats where file_name = '%s'", data.Pipeline, data.FileName)
	log.Info().Msg("stats sql query " + sqlstr)
	row := d.Connection.QueryRow(sqlstr)
	err := row.Scan(&recordsIn)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info().Msg("no rows in pipeline_stats yet")
		} else {
			return err
		}
	}

	recordsIn += data.RecordsIn
	sqlstr = fmt.Sprintf("UPSERT into %s.pipeline_stats (dataprov_id, file_name, records_in, lastupdated ) values ($1, $2, $3, 'now()')", data.Pipeline)
	log.Info().Msg("stats upsert " + sqlstr)
	upsertStmt, err := d.Connection.Prepare(sqlstr)
	if err != nil {
		return err
	}
	defer upsertStmt.Close()
	if _, err := upsertStmt.Exec(data.DataprovID, data.FileName, recordsIn); err != nil {
		return err
	}

	return nil
}
