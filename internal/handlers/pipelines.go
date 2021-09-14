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

	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/zerolog/log"

	"html/template"
	"net/http"
)

// PipelineInfo ....
type PipelineInfo struct {
	Name    string
	ID      string
	Running bool
}

// PipelinesPage ...
type PipelinesPage struct {
	UserEmail string
	List      []PipelineInfo
}

// Pipelines ...
func (u *HandlerWrapper) Pipelines(w http.ResponseWriter, r *http.Request) {

	pageValues := PipelinesPage{
		List:      make([]PipelineInfo, 0),
		UserEmail: u.UserEmail,
	}

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Some error")
		return
	}

	pipelineClient, err := pkg.NewClient(config, "")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Some error")
		return
	}

	pList, err := pipelineClient.List()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Some error")
		return
	}
	//	log.Info().Msg(fmt.Sprintf("all pipelines from kube %+v\n", pList))

	for i := 0; i < len(pList.Items); i++ {
		info := PipelineInfo{
			Name:    pList.Items[i].Name,
			ID:      pList.Items[i].Spec.Id,
			Running: u.getStatus(pList.Items[i].Name),
		}
		pageValues.List = append(pageValues.List, info)

	}

	log.Info().Msg(fmt.Sprintf("%d pipelines read\n", len(pageValues.List)))

	tmpl, err := template.ParseFiles("pages/home.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

func (u *HandlerWrapper) getStatus(pipelineName string) bool {
	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Some error")
		return false
	}

	req := pb.PingRequest{}

	_, err = client.Ping(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Some error")
		return false
	}

	return true
}
