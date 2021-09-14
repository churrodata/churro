// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package stats holds the pipeline stats logic
// for each pipeline as it executes, stats are written about
// each data source being processed, stats are per-pipeline, and
// the pipeline_stats table is created per-pipeline
// the intention of these stats is to give users insights into
// their pipeline processing
package stats

import (
	"time"
)

// PipelineStats ...
type PipelineStats struct {
	ID          int64
	DataprovID  string
	Pipeline    string
	FileName    string
	RecordsIn   int64
	LastUpdated time.Time
}
