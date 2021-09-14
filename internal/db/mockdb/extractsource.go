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

func (d MockChurroDatabase) UpdateExtractSourceMetric(a domain.ExtractSourceMetric) (err error) {

	return nil
}

func (d MockChurroDatabase) CreateExtractSourceMetric(a domain.ExtractSourceMetric) (err error) {

	return nil
}

func (d MockChurroDatabase) GetExtractSourceMetrics(id string) (wdirs []domain.ExtractSourceMetric, err error) {

	return wdirs, nil
}

func (d MockChurroDatabase) IsInitialized(tablename string) bool {
	return true
}
