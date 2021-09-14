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

import (
	"encoding/json"
	"fmt"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type Row struct {
	Fields []KV `json:"row"`
}
type IntermediateFormat struct {
	Path        string                   `json:"path"`
	Dataprov    string                   `json:"dataprov"`
	ColumnNames []string                 `json:"columnnames"`
	ColumnTypes []string                 `json:"columntypes"`
	Messages    []map[string]interface{} `json:"messages"`
}

// holds a single json message
type RawFormat struct {
	Path        string       `json:"path"`
	Dataprov    string       `json:"dataprov"`
	Columns     []Column     `json:"columns"`
	ColumnNames []string     `json:"columnnames"`
	ColumnTypes []string     `json:"columntypes"`
	Message     []byte       `json:"message"`
	Records     []GenericRow `json:"records"`
}

func (s IntermediateFormat) String() string {
	b, err := json.MarshalIndent(s, "", "")
	if err != nil {
		fmt.Printf("marshalling error %s\n", err.Error())
		return ""
	}
	return string(b)
}
