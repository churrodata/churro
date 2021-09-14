// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package mysql

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/stats"
	"github.com/churrodata/churro/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	DB_MYSQL = "mysql"
)

type MysqlChurroDatabase struct {
	Connection *sql.DB
	namespace  string
}

func (d MysqlChurroDatabase) GetVersion() (string, error) {
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

func (d MysqlChurroDatabase) CreateObjects(dbName string) error {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database " + dbName)

	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.pipelinemetric ( name varchar(30) PRIMARY KEY, value varchar(30) NOT NULL, lastupdated TIMESTAMP);", dbName)
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
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractsourcemetric ( extractsourceid varchar(30) primary key, name varchar(30) NOT NULL, value varchar(30) NOT NULL, lastupdated TIMESTAMP);", dbName)
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

func (d MysqlChurroDatabase) GetDatabaseType() string {
	return DB_MYSQL
}

func (d *MysqlChurroDatabase) GetConnection(creds config.DBCredentials, source v1alpha1.Source) (err error) {
	d.namespace = os.Getenv("CHURRO_NAMESPACE")
	if d.namespace == "" {
		log.Error().Stack().Msg("error CHURRO_NAMESPACE is empty")
		return fmt.Errorf("CHURRO_NAMESPACE env var required")
	}

	//TODO
	//mysqlConnectString := dbCreds.GetDBConnectString(source)
	//mysqlConnectString := "root:not-so-secure@tcp(localhost:3306)/test"
	mysqlConnectString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", source.Username, source.Password, source.Host, source.Port, source.Database)
	d.Connection, err = sql.Open("mysql", mysqlConnectString)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

// CreateDataprov inserts into the pipeline.dataprov table
func (d MysqlChurroDatabase) CreateDataprov(dp domain.DataProvenance) error {
	insertStmt, err := d.Connection.Prepare("insert into dataprov (id, name, path, lastupdated) values (?, ?, ?, now())")
	if err != nil {
		return err
	}
	if _, err := insertStmt.Exec(dp.ID, dp.Name, dp.Path); err != nil {
		return err
	}

	return nil
}

func (d *MysqlChurroDatabase) UpdatePipelineStats(t stats.PipelineStats) error {

	var recordsIn int64
	// get existing records count if a row exists
	row := d.Connection.QueryRow("SELECT records_in from pipeline_stats where file_name = ?", t.FileName)
	err := row.Scan(&recordsIn)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info().Msg("no rows in pipeline_stats yet")
		} else {
			return err
		}
	}
	recordsIn += t.RecordsIn

	var UPDATE = "UPDATE pipeline_stats set records_in = ?,  lastupdated = now() where file_name = ?"
	stmt, err := d.Connection.Prepare(UPDATE)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(recordsIn, t.FileName); err != nil {
		return err
	}

	return nil
}
