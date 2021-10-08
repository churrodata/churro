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
	"fmt"
	"net/http"

	"github.com/churrodata/churro/internal/extractsource"
	pb "github.com/churrodata/churro/rpc/extractsource"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// StartExtractSource ...
func (u *HandlerWrapper) StartExtractSource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	log.Info().Msg(fmt.Sprintf("StartExtractSource called...vars %+v\n", vars))
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	client, err := extractsource.GetExtractSourceServiceConnection(x.Name)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.StartAPIRequest{
		PipelineID:      pipelineID,
		ExtractSourceID: extractSourceID,
	}

	_, err = client.StartAPI(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	targetURL := fmt.Sprintf("/pipelines/%s", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// StopExtractSource ...
func (u *HandlerWrapper) StopExtractSource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	log.Info().Msg(fmt.Sprintf("StopExtractSource called...vars %+v\n", vars))
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	client, err := extractsource.GetExtractSourceServiceConnection(x.Name)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.StopAPIRequest{
		PipelineID:      pipelineID,
		ExtractSourceID: extractSourceID,
	}

	_, err = client.StopAPI(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	targetURL := fmt.Sprintf("/pipelines/%s", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}
