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
	"github.com/churrodata/churro/internal/domain"
)

func (s MockChurroDatabase) CreatePipelineDatabase(dbName string) error {

	return nil
}

func (s MockChurroDatabase) CreatePipelineObjects(dbName, username string) error {

	return nil
}

func (s MockChurroDatabase) GetAllPipelineMetrics() (metrics []domain.PipelineMetric, err error) {
	return metrics, nil
}

func (s MockChurroDatabase) UpdatePipelineMetric(m domain.PipelineMetric) error {

	return nil
}
func (s MockChurroDatabase) CreatePipelineMetric(m domain.PipelineMetric) error {

	return nil
}

func (s MockChurroDatabase) CreateUser(username, password string) (err error) {
	return nil
}

func (s MockChurroDatabase) CreateExtractLog(p domain.JobProfile) error {
	return nil

}
func (s MockChurroDatabase) UpdateExtractLog(p domain.JobProfile) error {
	return nil
}

func (s MockChurroDatabase) GetExtractLog(jobName string) (p domain.JobProfile, err error) {

	return p, nil

}

func (s MockChurroDatabase) GetExtractLogById(id string) (p domain.JobProfile, err error) {

	return p, nil

}
