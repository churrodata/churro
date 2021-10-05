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
	"fmt"

	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/zerolog/log"
)

func (d SinglestoreChurroDatabase) CreatePipelineObjects(dbName, username string) error {

	sqlStr := fmt.Sprintf("CREATE TABLE if not exists %s.dataprov ( id varchar(30) PRIMARY KEY, name varchar(30), path varchar(80), lastupdated TIMESTAMP);", dbName)
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

	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.pipeline_stats ( id serial PRIMARY KEY, dataprov_id text, file_name text, records_in bigint, lastupdated TIMESTAMP);", dbName)
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

	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractlog ( tablename varchar(32) not null, id varchar(32) PRIMARY KEY, dataprov_id varchar(32) not null, podname varchar(64) not null, poddate timestamp, records_loaded int not null, file_name varchar(64), lastupdated TIMESTAMP);", dbName)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	log.Info().Msg(sqlStr)

	return nil
}

/**
func (d SinglestoreChurroDatabase) UpdatePipeline(m domain.Pipeline) error {
	var UPDATE = fmt.Sprintf("UPDATE pipeline set lastupdated = now() where id = ?")
	stmt, err := d.Connection.Prepare(UPDATE)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	_, err = stmt.Exec(m.ID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
*/

/**
func (d SinglestoreChurroDatabase) GetAllPipelines() (pipes []domain.Pipeline, err error) {
	pipes = make([]domain.Pipeline, 0)

	rows, err := d.Connection.Query("SELECT id, name, lastupdated FROM pipeline")
	if err != nil {
		log.Error().Stack().Err(err)
		return pipes, err
	}

	for rows.Next() {
		p := domain.Pipeline{}
		err = rows.Scan(&p.ID, &p.Name, &p.LastUpdated)
		if err != nil {
			log.Error().Stack().Err(err)
			return pipes, err
		}
		pipes = append(pipes, p)
	}

	return pipes, nil
}
*/

/**
func (d SinglestoreChurroDatabase) GetPipeline(id string) (p domain.Pipeline, err error) {
	row := d.Connection.QueryRow("SELECT id,name, lastupdated FROM pipeline where id=?", id)
	switch err := row.Scan(&p.ID, &p.Name, &p.LastUpdated); err {
	case sql.ErrNoRows:
		log.Info().Msg("pipeline id was not found")
		return p, err
	case nil:
		log.Error().Msg("pipeline id was found")
		return p, nil
	default:
		return p, err
	}

	return p, nil
}
*/

/**
func (d SinglestoreChurroDatabase) DeletePipeline(id string) error {
	_, err := d.Connection.Exec(fmt.Sprintf("DELETE FROM pipeline where id='%s'", id))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
*/

/**
func (d SinglestoreChurroDatabase) CreatePipeline(p domain.Pipeline) (string, error) {
	p.ID = xid.New().String()
	insertStmt, err := d.Connection.Prepare("insert into pipeline(id, name, lastupdated) values(?,?,now())")
	if err != nil {
		log.Error().Stack().Err(err)
		return "", err
	}
	_, err = insertStmt.Exec(p.ID, p.Name)
	if err != nil {
		log.Error().Stack().Err(err)
		return "", err
	}

	return p.ID, nil
}
*/

func (d SinglestoreChurroDatabase) GetAllPipelineMetrics() (metrics []domain.PipelineMetric, err error) {
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
func (d SinglestoreChurroDatabase) UpdatePipelineMetric(m domain.PipelineMetric) error {
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
func (d SinglestoreChurroDatabase) CreatePipelineMetric(m domain.PipelineMetric) error {
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
func (d SinglestoreChurroDatabase) CreateUser(username, password string) error {

	// TODO need to implement 'if not exists' here!

	sqlStr := "create user " + username + " identified by '" + password + "';"
	stmt, err := d.Connection.Prepare(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		//return err
	}
	log.Info().Msg(sqlStr)

	sqlStr = "grant create,select,insert,delete,update on " + username + ".* to '" + username + "'@'%'"
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

func (d SinglestoreChurroDatabase) CreateExtractLog(p domain.JobProfile) error {
	insertStmt, err := d.Connection.Prepare("insert into extractlog(file_name, tablename, id, dataprov_id, podname, poddate, records_loaded, lastupdated) values(?,?,?,?,?,?,?,now())")
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}
	_, err = insertStmt.Exec(p.FileName, p.TableName, p.ID, p.DataProvenanceID, p.JobName, p.StartDate, p.RecordsLoaded)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
func (d SinglestoreChurroDatabase) UpdateExtractLog(p domain.JobProfile) error {
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

func (d SinglestoreChurroDatabase) GetExtractLog(jobName string) (p domain.JobProfile, err error) {
	row := d.Connection.QueryRow("SELECT tablename, id,dataprov_id, podname, poddate, records_loaded, file_name FROM extractlog where podname=?", jobName)
	err = row.Scan(&p.TableName, &p.ID, &p.DataProvenanceID, &p.JobName, &p.StartDate, &p.RecordsLoaded, &p.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg(jobName)
		return p, err
	}

	return p, nil
}

func (d SinglestoreChurroDatabase) GetExtractLogById(id string) (p domain.JobProfile, err error) {
	row := d.Connection.QueryRow("SELECT tablename, id,dataprov_id, podname, poddate, records_loaded, file_name FROM extractlog where id=?", id)
	err = row.Scan(&p.TableName, &p.ID, &p.DataProvenanceID, &p.JobName, &p.StartDate, &p.RecordsLoaded, &p.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg(id)
		return p, err
	}

	return p, nil
}
