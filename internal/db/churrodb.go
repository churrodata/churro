// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package db

import (
	"fmt"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db/cockroachdb"
	"github.com/churrodata/churro/internal/db/mockdb"
	"github.com/churrodata/churro/internal/db/mysql"
	"github.com/churrodata/churro/internal/db/singlestore"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/stats"

	"github.com/churrodata/churro/pkg/config"
)

// ChurroDatabase ...
type ChurroDatabase interface {
	GetConnection(creds config.DBCredentials, source v1alpha1.Source) error
	GetVersion() (string, error)
	CreateChurroDatabase(dbName string) error
	CreatePipelineDatabase(dbName string) error
	CreateObjects(dbName string) error
	CreatePipelineObjects(dbName, username string) error
	GetDatabaseType() string

	GetAllPipelineMetrics() ([]domain.PipelineMetric, error)
	UpdatePipelineMetric(m domain.PipelineMetric) error
	CreatePipelineMetric(m domain.PipelineMetric) error

	CreateExtractSourceMetric(m domain.ExtractSourceMetric) error
	UpdateExtractSourceMetric(m domain.ExtractSourceMetric) error
	GetExtractSourceMetrics(id string) ([]domain.ExtractSourceMetric, error)
	IsInitialized(tablename string) bool

	CreateUser(username, password string) error
	CreateTable(userid, dbname, tableName string, columnNames, columnTypes []string) error
	GetInsertStatement(scheme, database, tablename string, cols []string, vals []interface{}, key int64) error
	GetBulkInsertStatement(scheme, database, tableName string, cols []string, records []extractapi.GenericRow, colTypes []string) error

	UpdatePipelineStats(t stats.PipelineStats) error

	CreateDataprov(d domain.DataProvenance) error

	CreateAuthenticatedUser(u domain.AuthenticatedUser) error
	DeleteAuthenticatedUser(id string) error
	GetUserPipelineAccess(pipeline, id string) (domain.UserPipelineAccess, error)
	CreateAuthObjects() error
	CreateUserPipelineAccess(a domain.UserPipelineAccess) error
	UpdateUserPipelineAccess(a domain.UserPipelineAccess) error
	DeleteAllUserPipelineAccess(pipeline string) error
	DeleteUserPipelineAccess(pipeline, id string) error
	CreateUserProfile(u domain.UserProfile) error
	UpdateUserProfile(u domain.UserProfile) error
	DeleteUserProfile(id string) error
	Authenticate(email, password string) (domain.UserProfile, error)
	GetAllUserProfile() ([]domain.UserProfile, error)
	GetAllUserProfileForPipeline(pipeline string) ([]domain.UserProfile, error)
	GetUserProfileByEmail(email string) (domain.UserProfile, error)
	GetUserProfile(id string) (domain.UserProfile, error)
	Bootstrap() error
	CreateExtractLog(p domain.JobProfile) error
	UpdateExtractLog(p domain.JobProfile) error
	GetExtractLog(jobName string) (domain.JobProfile, error)
	GetExtractLogById(id string) (domain.JobProfile, error)
}

// NewChurroDB ...
func NewChurroDB(dbType string) (ChurroDatabase, error) {
	if dbType == domain.DatabaseCockroach {
		return &cockroachdb.CockroachChurroDatabase{}, nil
	}
	if dbType == domain.DatabaseMysql {
		return &mysql.MysqlChurroDatabase{}, nil
	}
	if dbType == domain.DatabaseSinglestore {
		return &singlestore.SinglestoreChurroDatabase{}, nil
	}
	if dbType == domain.DatabaseMock {
		return &mockdb.MockChurroDatabase{}, nil
	}
	return nil, fmt.Errorf("invalid churro database type specified")
}
