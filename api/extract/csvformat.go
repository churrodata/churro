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

const COLTYPE_TEXT = "TEXT"
const COLTYPE_VARCHAR = "VARCHAR(32)"
const COLTYPE_INT = "INT"
const COLTYPE_DECIMAL = "DECIMAL"

type CSVFormat struct {
	Path         string       `json:"path"`
	Dataprov     string       `json:"dataprov"`
	Tablename    string       `json:"tablename"`
	PipelineName string       `json:"pipelinename"`
	Columns      []Column     `json:"columns"`
	ColumnNames  []string     `json:"columnnames"`
	ColumnTypes  []string     `json:"columntypes"`
	Records      []GenericRow `json:"records"`
}
