// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package operator holds the churro operator logic
package operator

import (
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/pkg"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
)

func (r PipelineReconciler) processId(pipeline v1alpha1.Pipeline) error {

	if pipeline.Spec.Id == "" {
		log.Info().Msg("need pipeline ID for " + pipeline.Name)

		pipeline.Spec.Id = xid.New().String()

		// connect to the Kube API
		_, config, err := pkg.GetKubeClient()
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}

		pipelineClient, err := pkg.NewClient(config, pipeline.Name)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}

		_, err = pipelineClient.Update(&pipeline)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}
		log.Info().Msg("id updated for pipeline " + pipeline.Name)

	} else {
		log.Info().Msg("id already exists for pipeline " + pipeline.Name)
	}

	return nil
}
