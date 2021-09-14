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

import "github.com/churrodata/churro/internal/db"

// create the table based on column names and column types
func (s Server) tableCheck(columnNames, columnTypes []string) (err error) {

	userid := s.Pi.Spec.DataSource.Username
	dbname := s.Pi.Spec.DataSource.Database

	tableName := s.TableName

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return err
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		return err
	}

	err = churroDB.CreateTable(userid, dbname, tableName, columnNames, columnTypes)
	if err != nil {
		return err
	}

	return nil
}
