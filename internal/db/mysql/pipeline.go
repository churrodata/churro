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
	"fmt"

	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/zerolog/log"
)

func (d MysqlChurroDatabase) CreatePipelineDatabase(dbName string) error {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database " + dbName)
	return nil
}

func (d MysqlChurroDatabase) CreatePipelineObjects(dbName, username string) error {

	sqlStr := fmt.Sprintf("CREATE TABLE if not exists %s.dataprov ( id varchar(30) PRIMARY KEY, name varchar(30), path varchar(80), lastupdated TIMESTAMP default '1970-01-01 00:00:01');", dbName)
	stmt, err := d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}

	log.Info().Msg(sqlStr)

	// grant privs to pipeline database user
	// grant insert,update,select on pipeline1.dataprov to someuser
	sqlStr = fmt.Sprintf("grant insert,select on %s.dataprov to '%s';", dbName, username)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	log.Info().Msg(sqlStr)

	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.pipeline_stats ( id serial PRIMARY KEY, dataprov_id text, file_name text, records_in bigint, lastupdated TIMESTAMP default '1970-01-01 00:00:01');", dbName)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	log.Info().Msg(sqlStr)

	// grant privs to pipeline database user
	// grant insert,update,select on pipeline1.pipeline_stats to someuser
	sqlStr = fmt.Sprintf("grant insert,select on %s.pipeline_stats to '%s';", dbName, username)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	log.Info().Msg(sqlStr)

	sqlStr2 := fmt.Sprintf("CREATE TABLE if not exists %s.extractlog ( tablename varchar(32), id varchar(32) PRIMARY KEY, dataprov_id varchar(32), podname varchar(32), poddate timestamp, records_loaded bigint, file_name varchar(32), lastupdated TIMESTAMP default '1970-01-01 00:00:01');", dbName)
	stmt, err = d.Connection.Prepare(sqlStr2)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr2)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr2)
		return err
	}
	log.Info().Msg(sqlStr2)

	// grant privs to pipeline database user
	// grant insert,update,select on pipeline1.pipeline_stats to someuser
	sqlStr = fmt.Sprintf("grant insert,select on %s.extractlog to '%s';", dbName, username)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error on " + sqlStr)
		return err
	}
	log.Info().Msg(sqlStr)
	return nil
}

func (d MysqlChurroDatabase) GetAllPipelineMetrics() (metrics []domain.PipelineMetric, err error) {
	metrics = make([]domain.PipelineMetric, 0)

	rows, err := d.Connection.Query("SELECT name, value, lastupdated FROM pipelinemetric order by name")
	if err != nil {
		log.Error().Stack().Err(err)
		return metrics, err
	}

	for rows.Next() {
		p := domain.PipelineMetric{}
		err = rows.Scan(&p.Name, &p.Value, &p.LastUpdated)
		if err != nil {
			log.Error().Stack().Err(err)
			return metrics, err
		}
		metrics = append(metrics, p)
	}

	return metrics, nil
}
func (d MysqlChurroDatabase) UpdatePipelineMetric(m domain.PipelineMetric) error {
	var UPDATE = "UPDATE pipelinemetric set value = ?, lastupdated = now() where name = ?"

	stmt, err := d.Connection.Prepare(UPDATE)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	_, err = stmt.Exec(m.Value, m.Name)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
func (d MysqlChurroDatabase) CreatePipelineMetric(m domain.PipelineMetric) error {
	INSERT := "INSERT INTO pipelinemetric(name, value, lastupdated) values(?,?,now())"

	stmt, err := d.Connection.Prepare(INSERT)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}
	_, err = stmt.Exec(m.Name, m.Value)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
func (d MysqlChurroDatabase) CreateUser(username, password string) error {

	sqlStr := "create user if not exists '" + username + "'@'%' identified by '" + password + "';"
	stmt, err := d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	log.Info().Msg(sqlStr)

	sqlStr = "grant all privileges on " + username + ".* to '" + username + "'@'%' with grant option"
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	log.Info().Msg(sqlStr)

	return nil
}

func (d MysqlChurroDatabase) CreateExtractLog(p domain.JobProfile) error {
	insertStmt, err := d.Connection.Prepare("insert into extractlog(tablename, id, dataprov_id, podname, poddate, records_loaded, file_name, lastupdated) values(?,?,?,?,?,?,?,now())")
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}
	_, err = insertStmt.Exec(p.TableName, p.ID, p.DataProvenanceID, p.JobName, p.StartDate, p.RecordsLoaded, p.FileName)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
func (d MysqlChurroDatabase) UpdateExtractLog(p domain.JobProfile) error {
	var UPDATE = fmt.Sprintf("UPDATE extractlog set records_loaded = ?, lastupdated = now() where id = ?")
	stmt, err := d.Connection.Prepare(UPDATE)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	_, err = stmt.Exec(p.RecordsLoaded, p.ID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d MysqlChurroDatabase) GetExtractLog(jobName string) (p domain.JobProfile, err error) {
	row := d.Connection.QueryRow("SELECT tablename, id,dataprov_id, podname, poddate, records_loaded, file_name FROM extractlog where podname=?", jobName)
	err = row.Scan(&p.TableName, &p.ID, &p.DataProvenanceID, &p.JobName, &p.StartDate, &p.RecordsLoaded, &p.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting job from extractlog " + jobName)
		return p, err
	}

	return p, nil
}

func (d MysqlChurroDatabase) GetExtractLogById(id string) (p domain.JobProfile, err error) {
	row := d.Connection.QueryRow("SELECT tablename, id,dataprov_id, podname, poddate, records_loaded, file_name FROM extractlog where id=?", id)
	err = row.Scan(&p.TableName, &p.ID, &p.DataProvenanceID, &p.JobName, &p.StartDate, &p.RecordsLoaded, &p.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting job from extractlog by id" + id)
		return p, err
	}

	return p, nil
}
