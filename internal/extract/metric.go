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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var filesProcessedMetric prometheus.Counter

func (s *Server) createMetric() {
	filesProcessedMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "churro_processed_files_totals",
		Help:        "the total number of processed files for this pipeline",
		ConstLabels: prometheus.Labels{"pipeline": s.Pi.Name},
	})

	filesProcessedMetric.Add(1)

}
