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
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	"github.com/churrodata/churro/rpc/ctl"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// PipelineMetric ...
type PipelineMetric struct {
	Name  string
	Value string
}

// PipelineStatusHandler ...
func (u *HandlerWrapper) PipelineStatusHandler(w http.ResponseWriter, r *http.Request) {

	// PipelineDetail is a wrapper around a bunch of pipeline data
	// we use this as a model for the web UI only
	type XPipelineStatus struct {
		UserEmail string
		ErrorText string
		ID        string
		Name      string
		Pipeline  v1alpha1.Pipeline
		Metrics   []PipelineMetric
		Jobs      []domain.JobProfile
	}

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	log.Info().Msg("pipeline status: id " + pipelineID)

	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionRead,
	}
	if !m.Authorized(u.DatabaseType) {
		w.Write([]byte("user not authorized to view this pipeline"))
		return
	}

	var err error
	pipelineDetail := XPipelineStatus{
		ID:        pipelineID,
		UserEmail: u.UserEmail,
	}

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pipelineClient, err := pkg.NewClient(config, "")
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pList, err := pipelineClient.List()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	pipelineDetail.Pipeline = x
	pipelineDetail.Name = x.Name

	// get the pipeline info from the pipeline's ctl service
	client, err := GetServiceConnection(x.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to pipeline ctl service")
		pipelineDetail.ErrorText = pipelineDetail.ErrorText + ", " + err.Error()
	} else {

		piReq := pb.GetPipelineStatusRequest{Namespace: pipelineDetail.Name}
		piResponse, err := client.GetPipelineStatus(context.Background(), &piReq)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		pipelineDetail.Metrics = make([]PipelineMetric, 0)
		for i := 0; i < len(piResponse.Metrics); i++ {
			pm := PipelineMetric{
				Name:  piResponse.Metrics[i].GetName(),
				Value: piResponse.Metrics[i].GetValue(),
			}
			pipelineDetail.Metrics = append(pipelineDetail.Metrics, pm)
		}
		pipelineDetail.Jobs = make([]domain.JobProfile, 0)
		for i := 0; i < len(piResponse.Jobs); i++ {
			pm := domain.JobProfile{
				JobName:       piResponse.Jobs[i].Name,
				Status:        piResponse.Jobs[i].Status,
				DataSource:    piResponse.Jobs[i].Datasource,
				FileName:      piResponse.Jobs[i].FileName,
				TableName:     piResponse.Jobs[i].TableName,
				RecordsLoaded: int(piResponse.Jobs[i].RecordsLoaded),
				CompletedDate: piResponse.Jobs[i].CompletedDate,
				StartDate:     piResponse.Jobs[i].StartDate,
			}
			pipelineDetail.Jobs = append(pipelineDetail.Jobs, pm)
		}

	}

	tmpl, err := template.ParseFiles("pages/pipeline-status.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout", pipelineDetail)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// PipelineStatusJobLogHandler ...
func (u *HandlerWrapper) PipelineStatusJobLogHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	jobName := vars["jobname"]
	log.Info().Msg(fmt.Sprintf("pipeline status job log: id %s job %s\n", pipelineID, jobName))

	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionRead,
	}
	if !m.Authorized(u.DatabaseType) {
		w.Write([]byte("user not authorized to view this pipeline"))
		return
	}

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pipelineClient, err := pkg.NewClient(config, "")
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pList, err := pipelineClient.List()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	log.Info().Msg("pipeline name is " + x.Name)

	tmpl, err := template.ParseFiles("pages/pipeline-status-joblog.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	type JobLog struct {
		Log          string
		ID           string
		JobName      string
		PipelineName string
		UserEmail    string
		ErrorText    string
	}

	jobLogDetail := JobLog{
		UserEmail:    u.UserEmail,
		ID:           pipelineID,
		ErrorText:    "",
		JobName:      jobName,
		PipelineName: x.Name,
	}

	piReq := pb.GetPipelineJobLogRequest{
		Namespace: x.Name,
		Podname:   jobName,
	}
	client, err := GetServiceConnection(x.Name)
	if err != nil {
		log.Error().Stack().Msg("error connecting to pipeline ctl service ")
		jobLogDetail.ErrorText = jobLogDetail.ErrorText + ", " + err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", jobLogDetail)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	piResponse, err := client.GetPipelineJobLog(context.Background(), &piReq)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	jobLogDetail.Log = piResponse.Logstring

	//log.Info().Msg(fmt.Sprintf("joblog details are %+v", jobLogDetail))

	err = tmpl.ExecuteTemplate(w, "layout", jobLogDetail)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// return a map with the key being the pod name, and the value being
// the access value for that user Id
func getSelectedJobs(form url.Values) (values []string) {
	values = make([]string, 0)
	for k := range form {
		if strings.HasPrefix(k, "job-") {
			jobName := strings.TrimPrefix(k, "job-")
			values = append(values, jobName)
		}
	}

	return values
}

// DeleteJobs ...
func (u *HandlerWrapper) DeleteJobs(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	pipelineID := r.Form["pipelineid"][0]
	pipelineName := r.Form["pipelinename"][0]
	jobs := getSelectedJobs(r.Form)
	log.Info().Msg(fmt.Sprintf("selected jobs %+v pipelineID %s pipelineName %s", jobs, pipelineID, pipelineName))

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	piReq := pb.DeleteJobsRequest{Namespace: pipelineName}
	piReq.Jobs = make([]*ctl.PipelineJobStatus, 0)
	for i := 0; i < len(jobs); i++ {
		m1 := pb.PipelineJobStatus{}
		m1.Name = jobs[i]
		piReq.Jobs = append(piReq.Jobs, &m1)
	}

	_, err = client.DeleteJobs(context.Background(), &piReq)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/pipelines/%s#nav-jobs", pipelineID), 302)

}
