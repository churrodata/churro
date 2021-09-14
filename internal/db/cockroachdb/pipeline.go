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
	"time"

	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/zerolog/log"
)

func (s CockroachChurroDatabase) CreatePipelineDatabase(dbName string) error {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := s.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database" + dbName)

	// for cockroach we need to grant privs to the pipeline user on the
	// pipeline db

	// grant all on database c1 to c1
	sqlStr = fmt.Sprintf("grant all on database %s to %s", dbName, dbName)
	_, err = s.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database" + dbName)

	return nil
}

func (s CockroachChurroDatabase) CreatePipelineObjects(dbName, username string) error {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE TABLE if not exists %s.dataprov ( id STRING PRIMARY KEY, name STRING, path STRING, lastupdated TIMESTAMP);", dbName)
	stmt, err := s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	log.Info().Msg(sqlStr)

	// grant privs to pipeline database user
	// grant insert,update,select on pipeline1.dataprov to someuser
	sqlStr = fmt.Sprintf("grant insert,select on %s.dataprov to %s;", dbName, username)
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg(sqlStr)

	// grant privs to pipeline database user
	// grant create on pipelinedatabase to someuser
	sqlStr = fmt.Sprintf("grant all on database %s to %s;", dbName, username)
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg(sqlStr)

	/**
	  CREATE TABLE if not exists pipeline1.pipeline_stats (
	          id serial PRIMARY KEY,
	          dataprov_id bigint,
	          file_name text,
	          records_in bigint,
	          lastUpdated TIMESTAMP);
	*/
	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.pipeline_stats ( id serial PRIMARY KEY, dataprov_id text, file_name text, records_in bigint, lastupdated TIMESTAMP);", dbName)
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg(sqlStr)

	// grant privs to pipeline database user
	// grant insert,update,select on pipeline1.pipeline_stats to someuser
	sqlStr = fmt.Sprintf("grant insert,select on %s.pipeline_stats to %s;", dbName, username)
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg(sqlStr)

	sqlStr = fmt.Sprintf("CREATE TABLE if not exists %s.extractlog ( tablename string not null, id STRING PRIMARY KEY, dataprov_id STRING not null, podname STRING not null, poddate timestamp, records_loaded int not null, file_name string, lastupdated TIMESTAMP);", dbName)
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	log.Info().Msg(sqlStr)

	// grant privs to pipeline database user
	// grant insert,update,select on pipeline1.extractlog to someuser
	sqlStr = fmt.Sprintf("grant insert,select on %s.extractlog to %s;", dbName, username)
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Info().Msg(sqlStr)

	// TODO create an index on the stats table

	// update the churro.pipeline admin table for this new pipeline
	/**
	  insertStmt := "insert into churro.pipeline (name, config, lastupdated) values ($1, $2, now())"
	  if err != nil {
	      return err
	  }
	  _, err = db.Exec(insertStmt, cfg.PipelineName, cfg.String())
	  if err != nil {
	      return err
	  }
	*/

	return nil
}

func (s CockroachChurroDatabase) GetAllPipelineMetrics() (metrics []domain.PipelineMetric, err error) {
	metrics = make([]domain.PipelineMetric, 0)

	rows, err := s.Connection.Query("SELECT name, value, lastupdated FROM pipelinemetric order by name")
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

func (s CockroachChurroDatabase) UpdatePipelineMetric(m domain.PipelineMetric) error {
	datetime := time.Now()
	var UPDATE = fmt.Sprintf("UPDATE pipelinemetric set (value, lastupdated) = ('%s', '%v') where name = '%s'", m.Value, datetime.Format("2006-01-02T15:04:05.999999999"), m.Name)
	log.Info().Msg(UPDATE)

	_, err := s.Connection.Exec(UPDATE)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}
func (s CockroachChurroDatabase) CreatePipelineMetric(m domain.PipelineMetric) error {
	INSERT := "INSERT INTO pipelinemetric(name, value, lastupdated) values($1,$2,now()) returning name"

	var name string
	err := s.Connection.QueryRow(INSERT, m.Name, m.Value).Scan(&name)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (s CockroachChurroDatabase) CreateUser(username, password string) (err error) {
	sqlStr := fmt.Sprintf("create user if not exists %s;", username)
	var stmt *sql.Stmt
	stmt, err = s.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	return nil
}

func (s CockroachChurroDatabase) CreateExtractLog(p domain.JobProfile) error {
	var INSERT = "INSERT INTO extractlog ( tablename, id, dataprov_id, podname, poddate, records_loaded, file_name, lastupdated ) values ($1, $2, $3, $4, $5, $6, $7, now()) returning id"

	err := s.Connection.QueryRow(INSERT, p.TableName, p.ID, p.DataProvenanceID, p.JobName, p.StartDate, p.RecordsLoaded, p.FileName).Scan(&p.ID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil

}
func (s CockroachChurroDatabase) UpdateExtractLog(p domain.JobProfile) error {
	//datetime := time.Now()
	var UPDATE = fmt.Sprintf("UPDATE extractlog set (records_loaded, lastupdated) = (%d, now()) where id = '%s'", p.RecordsLoaded, p.ID)
	log.Info().Msg(UPDATE)
	stmt, err := s.Connection.Prepare(UPDATE)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (s CockroachChurroDatabase) GetExtractLog(jobName string) (p domain.JobProfile, err error) {

	row := s.Connection.QueryRow("SELECT tablename, id,dataprov_id, podname, poddate, records_loaded, file_name FROM extractlog where podname=$1", jobName)
	err = row.Scan(&p.TableName, &p.ID, &p.DataProvenanceID, &p.JobName, &p.StartDate, &p.RecordsLoaded, &p.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting job from extract log " + jobName)
		return p, err
	}

	log.Info().Msg(fmt.Sprintf("returning extract log values of %+v", p))
	return p, nil

}

func (s CockroachChurroDatabase) GetExtractLogById(id string) (p domain.JobProfile, err error) {

	row := s.Connection.QueryRow("SELECT tablename, id,dataprov_id, podname, poddate, records_loaded, file_name FROM extractlog where id=$1", id)
	err = row.Scan(&p.TableName, &p.ID, &p.DataProvenanceID, &p.JobName, &p.StartDate, &p.RecordsLoaded, &p.FileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting job by id from extract log " + id)
		return p, err
	}
	log.Info().Msg(fmt.Sprintf("returning extract log values of %+v\n", p))

	return p, nil

}
