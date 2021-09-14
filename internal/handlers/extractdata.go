// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package handlers

import (
	"context"
	"net/http"

	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// DownloadExtract ...
func (u *HandlerWrapper) DownloadExtract(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	field1 := r.Form["field1"][0]
	log.Info().Msg("field 1 is " + field1)

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	log.Info().Msg("pipeline detail: id " + pipelineID)
	pipelineName := r.Form["pipelinename"][0]
	log.Info().Msg("pipeline detail: pipelinename " + pipelineName)

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		a := u.Copy(err.Error())
		a.ShowCreatePipeline(w, r)
		return
	}

	req := pb.GetExtractDataRequest{
		Namespace: pipelineName,
	}

	response, err := client.GetExtractData(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		a := u.Copy(err.Error())
		a.ShowCreatePipeline(w, r)
		return
	}

	w.Write(response.ExtractData)
}
