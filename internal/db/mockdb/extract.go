// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package mockdb

import (
	extractapi "github.com/churrodata/churro/api/extract"
)

func (d MockChurroDatabase) CreateTable(userid, dbname, tableName string, columnNames []string, columnTypes []string) (err error) {

	return nil
}

func getTableColumns(columnNames, columnTypes []string) string {
	return ""
}

func (d MockChurroDatabase) GetInsertStatement(scheme, database, tablename string, cols []string, vals []interface{}, primarykey int64) error {

	return nil
}

func (d MockChurroDatabase) GetBulkInsertStatement(scheme, database, tableName string, cols []string, records []extractapi.GenericRow, colTypes []string) error {

	return nil

}
