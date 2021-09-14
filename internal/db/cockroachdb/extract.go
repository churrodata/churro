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
	"strings"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/rs/zerolog/log"
)

func (d CockroachChurroDatabase) CreateTable(userid, dbname, tableName string, columnNames []string, columnTypes []string) (err error) {

	sqlStr := fmt.Sprintf("CREATE TABLE if not exists %s.%s ( primarykey bigint PRIMARY KEY, dataformat text, %s lastupdated TIMESTAMP);", dbname, tableName, getTableColumns(columnNames, columnTypes))
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

	log.Debug().Msg("Table created successfully.." + tableName)
	// grant privs to pipeline database user
	// grant insert,select on foo.churro,foo.dataprov to foo
	sqlStr = fmt.Sprintf("grant insert,select on %s.%s to %s;", dbname, tableName, userid)
	stmt, err = d.Connection.Prepare(sqlStr)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	log.Debug().Msg(sqlStr)

	return nil
}

func getTableColumns(columnNames, columnTypes []string) string {
	var result string
	for i, v := range columnNames {
		col := fmt.Sprintf("%s %s,", v, columnTypes[i])
		result = result + col
	}
	log.Debug().Msg("getTableColumns " + result)
	return result
}

func (d CockroachChurroDatabase) GetInsertStatement(scheme, database, tablename string, cols []string, vals []interface{}, primarykey int64) error {

	var b string
	for _, v := range cols {
		b = b + fmt.Sprintf("%s,", v)
	}
	var c string
	for _, v := range vals {
		c = c + fmt.Sprintf("'%s',", v)
	}

	csvsql := fmt.Sprintf("insert into %s.%s (primarykey, dataformat, %s lastupdated) values (%d, '%s', %s now())", database, tablename, b, primarykey, scheme, c)

	_, err := d.Connection.Query(csvsql)
	if err != nil {
		log.Error().Stack().Err(err).Msg(csvsql)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) GetBulkInsertStatement(scheme, database, tableName string, cols []string, records []extractapi.GenericRow, colTypes []string) error {

	var sqlString strings.Builder

	var colNames string
	for i, v := range cols {
		if i < len(cols)-1 {
			colNames = colNames + fmt.Sprintf("%s,", v)
		} else {
			colNames = colNames + fmt.Sprintf("%s", v)
		}
	}
	tmp := fmt.Sprintf("insert into %s.%s (primarykey, dataformat, %s, lastupdated) values ", database, tableName, colNames)
	sqlString.WriteString(tmp)
	numRecords := len(records)

	var currentRecord int
	for _, r := range records {
		//log.Info().Msg(fmt.Sprintf("jeff colTypes %v\n", colTypes))
		//log.Info().Msg(fmt.Sprintf("jeff cols %v\n", cols))
		//log.Info().Msg(fmt.Sprintf("jeff r.Cols %v\n", r.Cols))
		currentRecord++
		var tmp string
		var colValues string
		for i := 0; i < len(r.Cols); i++ {
			if i == len(r.Cols)-1 {
				if colTypes[i] == extractapi.COLTYPE_TEXT ||
					colTypes[i] == extractapi.COLTYPE_VARCHAR {
					colValues = colValues + "'" + r.Cols[i].(string) + "'"
				} else {
					colValues = colValues + fmt.Sprintf("%v", r.Cols[i])
				}
			} else {
				if colTypes[i] == extractapi.COLTYPE_TEXT ||
					colTypes[i] == extractapi.COLTYPE_VARCHAR {
					colValues = colValues + "'" + r.Cols[i].(string) + "'" + ","
				} else {
					colValues = colValues + fmt.Sprintf("%v", r.Cols[i]) + ","
				}
			}
		}
		if currentRecord < numRecords {
			tmp = fmt.Sprintf("(%d, '%s', %s, now()),", r.Key, scheme, colValues)
		} else {
			tmp = fmt.Sprintf("(%d, '%s', %s, now())", r.Key, scheme, colValues)
		}
		sqlString.WriteString(tmp)
	}

	log.Info().Msg(sqlString.String())

	_, err := d.Connection.Exec(sqlString.String())
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil

}
