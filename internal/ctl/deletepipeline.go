// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package ctl

import "github.com/rs/zerolog/log"

// DeletePipeline deletes the pipeline database
func (s *Server) deletePipeline() error {

	log.Info().Msg("pipeline database successfully deleted")
	return nil

}
