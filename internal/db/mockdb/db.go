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
	"database/sql"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/stats"
	"github.com/churrodata/churro/pkg/config"
)

const (
	DB_MOCK = "mockdb"
)

type MockChurroDatabase struct {
	Connection *sql.DB
	namespace  string
}

func (d MockChurroDatabase) GetVersion() (string, error) {
	return "mock-1.0", nil
}

func (d MockChurroDatabase) CreateObjects(dbName string) error {

	return nil
}
func (d MockChurroDatabase) GetDatabaseType() string {
	return DB_MOCK
}

func (d *MockChurroDatabase) GetConnection(dbCreds config.DBCredentials, source v1alpha1.Source) (err error) {

	return nil
}

func (d MockChurroDatabase) CreateDataprov(data domain.DataProvenance) error {
	return nil
}

func (d *MockChurroDatabase) UpdatePipelineStats(data stats.PipelineStats) error {
	return nil
}
