// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package ctl

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"

	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TableInfo defines the ...
type TableInfo struct {
	Name        string
	Columns     []string
	ColumnTypes []string
	Create      string
	Select      string
	Insert      string
}

// ExtractData defines the ...
type ExtractData struct {
	MaxRows         int
	DBPath          string
	PGConnectString string
	SourceDB        *sql.DB
	DB              *sql.DB
	Tables          []TableInfo
	Namespace       string
}

// GetExtractData ...
func (s *Server) GetExtractData(ctx context.Context, request *pb.GetExtractDataRequest) (response *pb.GetExtractDataResponse, err error) {

	response = &pb.GetExtractDataResponse{}

	if request.Namespace == "" {
		return nil, status.Errorf(codes.InvalidArgument,
			"namespace is required")
	}

	// create sqlite db
	ed := ExtractData{
		MaxRows:   10,
		Namespace: request.Namespace,
		Tables:    make([]TableInfo, 0),
	}

	ed.PGConnectString = s.DBCreds.GetDBConnectString(s.Pi.Spec.DataSource)
	log.Info().Msg("GetExtractData pgConnectString " + ed.PGConnectString)
	ed.SourceDB, err = sql.Open("postgres", ed.PGConnectString)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = ed.Initialize()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = ed.Initialize()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	defer os.Remove(ed.DBPath)

	// for each table in cockroach
	err = ed.GetTables()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// select limit output to 10 rows per table
	// select * from foo.dataprov order by lastupdated desc limit 10

	// create table in sqlite
	err = ed.CreateTables()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	for t := 0; t < len(ed.Tables); t++ {
		ed.Tables[t].Print()
	}

	err = ed.CopyData()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// zip up sqlite file
	var zBytes []byte
	zBytes, err = ed.ZipIt()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	response.ExtractData = zBytes
	return response, nil
}

// Initialize ...
func (s *ExtractData) Initialize() (err error) {
	file, err := ioutil.TempFile("/tmp", "extract.*.db")
	if err != nil {
		return err
	}
	log.Info().Msg("created " + file.Name())
	s.DBPath = file.Name()

	log.Info().Msg("pg connect string " + s.PGConnectString)

	s.SourceDB, err = sql.Open("postgres", s.PGConnectString)

	if err != nil {
		return err
	}
	s.DB, err = sql.Open("sqlite3", s.DBPath)
	if err != nil {
		return err
	}
	return err
}

// GetTables ...
func (s *ExtractData) GetTables() (err error) {

	// "show tables from pipeline1"
	rows, err := s.SourceDB.Query("show tables from " + s.Namespace)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	log.Info().Msg("tables")
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}
		log.Info().Msg(tableName)

		cRows, err := s.SourceDB.Query(fmt.Sprintf("select column_name, data_type from information_schema.columns where column_name != 'rowid' and table_name = '%s'", tableName))
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}

		log.Info().Msg("columns")
		columns := make([]string, 0)
		columnTypes := make([]string, 0)
		for cRows.Next() {
			var columnName, dataType string
			if err := cRows.Scan(&columnName, &dataType); err != nil {
				return err
			}
			log.Info().Msg("column " + columnName + dataType)
			columns = append(columns, columnName)
			columnTypes = append(columnTypes, dataType)
		}
		cRows.Close()
		// table_name
		t1 := TableInfo{
			Name:        tableName,
			Columns:     columns,
			ColumnTypes: columnTypes,
		}
		t1.getCreateSQL()
		t1.getSelectSQL(s.Namespace, s.MaxRows)
		t1.getInsertSQL()
		s.Tables = append(s.Tables, t1)
	}
	rows.Close()

	return nil
}

// CreateTables ...
func (s ExtractData) CreateTables() (err error) {
	for i := 0; i < len(s.Tables); i++ {
		// "show columns from pipeline1.dataprov"
		// column_name, data_type
		log.Info().Msg(s.Tables[i].Create)
		_, err := s.DB.Exec(s.Tables[i].Create)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TableInfo) getCreateSQL() {
	str := "create table " + s.Name + " ("
	for i := 0; i < len(s.Columns); i++ {
		str = str + fmt.Sprintf("%s %s", s.Columns[i], s.ColumnTypes[i])
		if i != len(s.Columns)-1 {
			str = str + ","
		}
	}
	str = str + ")"
	s.Create = str
}

func (s *TableInfo) getSelectSQL(namespace string, maxRows int) {
	str := "select "
	for i := 0; i < len(s.Columns); i++ {
		if s.Columns[i] == "lastupdated" {
			str = str + fmt.Sprintf("%s::text ", s.Columns[i])
		} else {
			str = str + fmt.Sprintf("%s ", s.Columns[i])
		}
		if i != len(s.Columns)-1 {
			str = str + ","
		}
	}
	str = str + " from " + namespace + "." + s.Name
	str = str + " order by lastupdated desc limit " + strconv.Itoa(maxRows)
	s.Select = str
}
func (s *TableInfo) getInsertSQL() {
	str := "insert into " + s.Name + " ("
	for i := 0; i < len(s.Columns); i++ {
		str = str + fmt.Sprintf("%s ", s.Columns[i])
		if i != len(s.Columns)-1 {
			str = str + ","
		}
	}
	str = str + ") values ("
	for i := 0; i < len(s.Columns); i++ {
		str = str + "?"
		if i != len(s.Columns)-1 {
			str = str + ","
		}
	}
	str = str + ")"
	s.Insert = str
}

// Print ...
func (s *TableInfo) Print() {
	log.Info().Msg("\nTable")
	log.Info().Msg("INSERT: " + s.Insert)
	log.Info().Msg("SELECT: " + s.Select)
	log.Info().Msg("CREATE: " + s.Create)
	for i := 0; i < len(s.Columns); i++ {
		log.Info().Msg("Column: " + s.Columns[i] + s.ColumnTypes[i])
	}
}

// ZipIt ...
func (s *ExtractData) ZipIt() (b []byte, err error) {
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	content, err := ioutil.ReadFile(s.DBPath)
	if err != nil {
		return b, err
	}

	var files = []struct {
		Name string
		Body []byte
	}{
		{"extract.db", content},
	}

	for _, file := range files {
		zipFile, err := zipWriter.Create(file.Name)
		if err != nil {
			return b, err
		}
		_, err = zipFile.Write(file.Body)
		if err != nil {
			return b, err
		}
	}

	err = zipWriter.Close()
	if err != nil {
		return b, err
	}

	return buf.Bytes(), nil
}

// CopyData ...
func (s ExtractData) CopyData() (err error) {

	for t := 0; t < len(s.Tables); t++ {
		table := s.Tables[t]
		// build the prepared statement for this sqlite table
		insertStmt, err := s.DB.Prepare(table.Insert)
		if err != nil {
			return err
		}
		// run select statement on this table in cockroach db
		log.Info().Msg("would run " + table.Select)
		rows, err := s.SourceDB.Query(table.Select)
		if err != nil {
			return err
		}
		columns, err := rows.Columns()
		colNum := len(columns)

		var values = make([]interface{}, colNum)
		for i := range values {
			var ii interface{}
			values[i] = &ii
		}
		for rows.Next() {
			err := rows.Scan(values...)
			if err != nil {
				return err
			}
			var colValues = make([]interface{}, colNum)

			for i, colName := range columns {
				var rawValue = *(values[i].(*interface{}))
				var rawType = reflect.TypeOf(rawValue)

				log.Info().Msg(fmt.Sprintf("%s %s %s", colName, rawType, rawValue))
				colValues[i] = &rawValue
			}
			_, err = insertStmt.Exec(colValues...)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
