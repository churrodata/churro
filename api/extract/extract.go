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

const (
	XLSXScheme     = "xlsx"
	CSVScheme      = "csv"
	XMLScheme      = "xml"
	JSONScheme     = "json"
	JSONPathScheme = "jsonpath"
	FinnHubScheme  = "finnhub-stocks"
	APIScheme      = "api"
	HTTPPostScheme = "httppost"
)

type LoaderMessage struct {
	Key        int64  `json:"key"` // only used for JSON messages
	Metadata   []byte `json:"metadata"`
	DataFormat string `json:"dataformat"`
}
