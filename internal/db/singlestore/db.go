// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package singlestore

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
	DB_SINGLESTORE = "singlestore"
)

type SinglestoreChurroDatabase struct {
	Connection *sql.DB
	namespace  string
}

func (d SinglestoreChurroDatabase) GetVersion() (string, error) {
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

func (d SinglestoreChurroDatabase) CreatePipelineDatabase(dbName string) error {
	// make sure pipeline database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database " + dbName)
	return nil
}

func (d SinglestoreChurroDatabase) CreateObjects(dbName string) error {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database " + dbName)

	/**
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.watchdirectory ( id varchar(30), name varchar(30) NOT NULL, path varchar(30) NOT NULL, scheme varchar(30) NOT NULL, regex varchar(30) NOT NULL, cronexpression varchar(30), tablename varchar(30) NOT NULL, lastupdated TIMESTAMP, shard key (id), primary key (id, path));", dbName)

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
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractrule ( id varchar(30) PRIMARY KEY, watchdirectoryid varchar(30) NOT NULL, columnname varchar(30) NOT NULL, columnpath varchar(30) NOT NULL, columntype varchar(30) NOT NULL, matchvalues varchar(30), transformfunction varchar(30), lastupdated TIMESTAMP);", dbName)
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
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extension ( id varchar(30) PRIMARY KEY, watchdirectoryid varchar(30) NOT NULL, extensionname varchar(30) NOT NULL, extensionpath varchar(80) NOT NULL, lastupdated TIMESTAMP);", dbName)
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
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.transformfunction ( id varchar(30) PRIMARY KEY, name varchar(30) NOT NULL, source tinytext NOT NULL, lastupdated TIMESTAMP);", dbName)
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
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractsourcemetric ( extractsourceid varchar(30) NOT NULL, name varchar(30) NOT NULL, value varchar(30) NOT NULL, lastupdated TIMESTAMP );", dbName)
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

func (d SinglestoreChurroDatabase) GetDatabaseType() string {
	return DB_SINGLESTORE
}

func (d *SinglestoreChurroDatabase) GetConnection(creds config.DBCredentials, source v1alpha1.Source) (err error) {
	d.namespace = os.Getenv("CHURRO_NAMESPACE")
	if d.namespace == "" {
		log.Error().Stack().Msg("error CHURRO_NAMESPACE is empty")
		return fmt.Errorf("CHURRO_NAMESPACE env var required")
	}

	//TODO
	//mysqlConnectString := dbCreds.GetDBConnectString(source)
	//mysqlConnectString := "root:not-so-secure@tcp(localhost:3306)/test"
	mysqlConnectString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", source.Username, source.Password, source.Host, source.Port, source.Database)
	log.Info().Msg("connection string " + mysqlConnectString)
	d.Connection, err = sql.Open("mysql", mysqlConnectString)
	if err != nil {
		log.Info().Msg("error in GetConnection " + err.Error())
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) CreateDataprov(dp domain.DataProvenance) error {
	insertStmt, err := d.Connection.Prepare("insert into dataprov (id, name, path, lastupdated) values (?, ?, ?, now())")
	if err != nil {
		return err
	}
	if _, err := insertStmt.Exec(dp.ID, dp.Name, dp.Path); err != nil {
		return err
	}

	return nil
}

func (d *SinglestoreChurroDatabase) UpdatePipelineStats(t stats.PipelineStats) error {

	return nil
}
