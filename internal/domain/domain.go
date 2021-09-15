// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package domain

import "time"

// DatabaseCockroach ...
const DatabaseCockroach = "cockroachdb"

// DatabaseMysql ...
const DatabaseMysql = "mysql"

// DatabaseSinglestore ...
const DatabaseSinglestore = "singlestore"

// DatabaseMock ...
const DatabaseMock = "mockdb"

// MetricFilesProcessed ...
const MetricFilesProcessed = "Files Processed"

// MetricLastFileProcessed ...
const MetricLastFileProcessed = "Last File Processed"

// Extension ...
type Extension struct {
	ID              string    `json:"id"`
	ExtractSourceID string    `json:"extractsourceid"`
	ExtensionName   string    `json:"extensionname"`
	ExtensionPath   string    `json:"extensionpath"`
	LastUpdated     time.Time `json:"lastupdated"`
}

// ExtractRule ...
type ExtractRule struct {
	ID                string    `json:"id"`
	ExtractSourceID   string    `json:"extractsourceid"`
	ColumnName        string    `json:"columnname"`
	ColumnPath        string    `json:"columnpath"`
	ColumnType        string    `json:"columntype"`
	MatchValues       string    `json:"matchvalues"`
	TransformFunction string    `json:"transformfunction"`
	LastUpdated       time.Time `json:"lastupdated"`
}

// ExtractSource ....
type ExtractSource struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	Scheme         string `json:"scheme"`
	Regex          string `json:"regex"`
	Tablename      string `json:"tablename"`
	Cronexpression string `json:"cronexpression"`
	Skipheaders    int    `json:"skipheaders"`
	Multiline      bool   `json:"multiline"`
	Sheetname      string `json:"sheetname"`
	// Initialized is calculated, not persisted
	Initialized  bool                   `json:"initialized"`
	Running      bool                   `json:"running"`
	ExtractRules map[string]ExtractRule `json:"extractrules"`
	Extensions   map[string]Extension   `json:"extensions"`
	LastUpdated  time.Time              `json:"lastupdated"`
}

// ExtractSourceMetric ...
type ExtractSourceMetric struct {
	ExtractSourceID string    `json:"extractsourceid"`
	Name            string    `json:"name"`
	Value           string    `json:"value"`
	LastUpdated     time.Time `json:"lastupdated"`
}

// DataProvenance ...
type DataProvenance struct {
	ID          string
	Name        string
	Path        string
	LastUpdated time.Time
}

// Pipeline ...
type Pipeline struct {
	ID          string
	Name        string
	LastUpdated string
}

// PipelineMetric ...
type PipelineMetric struct {
	Name        string
	Value       string
	LastUpdated string
}

// AuthenticatedUser authenticated users
type AuthenticatedUser struct {
	ID          string    `json:"id"`
	Token       string    `json:"token"`
	Locked      bool      `json:"locked"`
	LastUpdated time.Time `json:"lastupdated"`
}

// UserProfile users are global to all the pipelines
type UserProfile struct {
	ID          string    `json:"id"`
	FirstName   string    `json:"firstname"`
	LastName    string    `json:"lastname"`
	Password    string    `json:"password"`
	Access      string    `json:"access"`
	Email       string    `json:"email"`
	LastUpdated time.Time `json:"lastupdated"`
}

// JobProfile ...
type JobProfile struct {
	ID               string `json:"id"`
	JobName          string `json:"jobname"`
	RecordsLoaded    int    `json:"recordsloaded"`
	DataProvenanceID string `json:"dpid"`
	DataSource       string `json:"datasource"`
	StartDate        string `json:"startdate"`
	CompletedDate    string `json:"completeddate"`
	FileName         string `json:"filename"`
	TableName        string `json:"tablename"`
	Status           string `json:"status"`
}

// UserPipelineAccess users can have access granted to a pipeline
type UserPipelineAccess struct {
	UserProfileID string    `json:"userprofileid"`
	PipelineID    string    `json:"pipelineid"`
	Access        string    `json:"access"`
	LastUpdated   time.Time `json:"lastupdated"`
}

// TransformFunction ...
type TransformFunction struct {
	ID          string    `json:"id"`
	Name        string    `json:"transformname"`
	Source      string    `json:"transformsource"`
	LastUpdated time.Time `json:"lastupdated"`
}
