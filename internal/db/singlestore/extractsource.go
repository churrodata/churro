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

func (d SinglestoreChurroDatabase) UpdateExtractSourceMetric(a domain.ExtractSourceMetric) error {
	var UPDATE = "UPDATE extractsourcemetric set value = ?, lastupdated = now() where extractsourceid = ? and name = ?"
	log.Info().Msg(UPDATE)

	stmt, err := d.Connection.Prepare(UPDATE)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}
	_, err = stmt.Exec(a.Value, a.ExtractSourceID, a.Name)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) CreateExtractSourceMetric(a domain.ExtractSourceMetric) (err error) {
	var INSERT = fmt.Sprintf("INSERT INTO extractsourcemetric(extractsourceid, name, value, lastupdated) values(?,?,?,now())")
	stmt, err := d.Connection.Prepare(INSERT)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	_, err = stmt.Exec(a.ExtractSourceID, a.Name, a.Value)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) GetExtractSourceMetrics(id string) (wdirs []domain.ExtractSourceMetric, err error) {
	rows, err := d.Connection.Query(fmt.Sprintf("SELECT extractsourceid, name, value from extractsourcemetric where extractsourceid = '%s'", id))
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return wdirs, err
	}

	for rows.Next() {
		p := domain.ExtractSourceMetric{}
		err = rows.Scan(&p.ExtractSourceID, &p.Name, &p.Value)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return wdirs, err
		}
		wdirs = append(wdirs, p)
	}

	return wdirs, nil
}

func (d SinglestoreChurroDatabase) GetExtractSourceByName(name string) (a domain.ExtractSource, err error) {
	row := d.Connection.QueryRow("SELECT id, cronexpression, tablename, path, scheme, regex, lastupdated FROM extractsource where name=?", name)
	err = row.Scan(&a.ID, &a.Cronexpression, &a.Tablename, &a.Path, &a.Scheme, &a.Regex, &a.LastUpdated)
	if err != nil {
		log.Info().Msg("extractsource name was not found")
		return a, err
	}

	a.Name = name

	a.Initialized = d.IsInitialized(a.Tablename)

	return a, nil
}

func (d SinglestoreChurroDatabase) IsInitialized(tablename string) bool {
	sqlString := fmt.Sprintf("select count(*) from %s.%s", d.namespace, tablename)
	row := d.Connection.QueryRow(sqlString)
	var t int
	err := row.Scan(&t)
	if err != nil {
		log.Error().Stack().Err(err)
		return false
	}
	return true
}
