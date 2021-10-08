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
	"archive/zip"
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/pipeline"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// PipelineDetailHandler ...
func (u *HandlerWrapper) PipelineDetailHandler(w http.ResponseWriter, r *http.Request) {

	// PipelineDetail is a wrapper around a bunch of pipeline data
	// we use this as a model for the web UI only
	type XPipelineDetail struct {
		UserEmail          string
		ErrorText          string
		ID                 string
		Name               string
		DBConsoleURL       string
		Pipeline           v1alpha1.Pipeline
		Metrics            []PipelineMetric
		TransformFunctions []domain.TransformFunction
		Users              []domain.UserProfile
		Jobs               []domain.JobProfile
		ExtractSources     []domain.ExtractSource
	}

	vars := mux.Vars(r)
	pipelineID := vars["id"]

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
	pipelineDetail := XPipelineDetail{
		ID:        pipelineID,
		UserEmail: u.UserEmail,
	}

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	// get the pipeline users

	usersList, err := churroDB.GetAllUserProfileForPipeline(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	jobsList := make([]domain.JobProfile, 0)
	jp := domain.JobProfile{
		JobName:       "somejob",
		Status:        "running",
		DataSource:    "mycsvdatasource",
		CompletedDate: "10-10-2021 8:00pm",
		StartDate:     "10-10-2021 7:00pm",
	}
	jobsList = append(jobsList, jp)

	pipelineDetail.Jobs = jobsList
	pipelineDetail.Users = usersList

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pipelineDetail.Pipeline = x
	pipelineDetail.Name = x.Name

	// get the pipeline info from the pipeline's ctl service
	client, err := GetServiceConnection(x.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to pipeline ctl service")
		pipelineDetail.ErrorText = pipelineDetail.ErrorText + ", " + err.Error()
	} else {
		gtfReq := pb.GetTransformFunctionsRequest{Namespace: pipelineDetail.Name}
		gtfResponse, err := client.GetTransformFunctions(context.Background(), &gtfReq)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		byt := []byte(gtfResponse.FunctionsString)
		var list []domain.TransformFunction
		err = json.Unmarshal(byt, &list)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		pipelineDetail.TransformFunctions = list

		piReq := pb.GetPipelineRequest{Namespace: pipelineDetail.Name}
		piResponse, err := client.GetPipeline(context.Background(), &piReq)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		log.Info().Msg("got back for db console url" + piResponse.DatabaseConsoleURL)
		pipelineDetail.DBConsoleURL = piResponse.DatabaseConsoleURL

		gwdRequest := pb.GetExtractSourcesRequest{Namespace: pipelineDetail.Name}

		gwdResponse, err := client.GetExtractSources(context.Background(), &gwdRequest)
		if err != nil {
			pipelineDetail.ErrorText = pipelineDetail.ErrorText + ", " + err.Error()
		}

		var extractSources []domain.ExtractSource
		err = json.Unmarshal([]byte(gwdResponse.ExtractSourcesString), &extractSources)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		pipelineDetail.ExtractSources = extractSources
	}

	piReq := pb.GetPipelineStatusRequest{Namespace: pipelineDetail.Name}
	piResponse, err := client.GetPipelineStatus(context.Background(), &piReq)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	// get metrics
	pipelineDetail.Metrics = make([]PipelineMetric, 0)
	for i := 0; i < len(piResponse.Metrics); i++ {
		pm := PipelineMetric{
			Name:  piResponse.Metrics[i].GetName(),
			Value: piResponse.Metrics[i].GetValue(),
		}
		pipelineDetail.Metrics = append(pipelineDetail.Metrics, pm)
	}

	// get jobs
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

	tmpl, err := template.ParseFiles("pages/pipeline-detail.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout", pipelineDetail)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// UpdatePipelineDetail ...
func (u *HandlerWrapper) UpdatePipelineDetail(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	pipelineID := r.Form["pipelineid"][0]

	if pipelineID == "" {
		a := u.Copy("invalid pipeline")
		a.PipelineDetailHandler(w, r)
		return
	}
	if r.Form["loaderpctheadroom"][0] == "" {
		a := u.Copy("invalid loader pct head room")
		a.PipelineDetailHandler(w, r)
		return
	}
	if r.Form["loaderqueuesize"][0] == "" {
		a := u.Copy("invalid loader queue size")
		a.PipelineDetailHandler(w, r)
		return
	}

	i, err := strconv.Atoi(r.Form["loaderqueuesize"][0])
	if err != nil {
		a := u.Copy("invalid loader queue size")
		a.PipelineDetailHandler(w, r)
		return
	}
	loaderQueueSize := i

	i, err = strconv.Atoi(r.Form["loaderpctheadroom"][0])
	if err != nil {
		a := u.Copy("invalid loader pct head room")
		a.PipelineDetailHandler(w, r)
		return
	}
	loaderPctHeadRoom := i

	log.Info().Msg(fmt.Sprintf("updating queuesize=%d headroom=%d\n", loaderQueueSize, loaderPctHeadRoom))
	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionWrite,
	}
	if !m.Authorized(u.DatabaseType) {
		a := u.Copy("user not authorized to update this pipeline")
		a.PipelineDetailHandler(w, r)
		return
	}

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pipelineClient, err := pkg.NewClient(config, x.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}
	_, err = pipelineClient.Update(&x)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineDetailHandler(w, r)
		return
	}

	http.Redirect(w, r, "/", 302)
}

// CreatePipelineDetail ...
func (u *HandlerWrapper) CreatePipelineDetail(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	p := v1alpha1.Pipeline{}

	p.ObjectMeta.Name = r.Form["pipelinename"][0]
	p.ObjectMeta.Labels = make(map[string]string)
	p.ObjectMeta.Labels["name"] = p.ObjectMeta.Name
	maxjobs := r.Form["maxjobs"][0]
	storageSize := r.Form["storagesize"][0]
	storageClassName := r.Form["storageclassname"][0]
	accessMode := r.Form["accessmode"][0]
	dbPassword := r.Form["dbpassword"][0]
	dbPassword2 := r.Form["dbpassword2"][0]
	log.Info().Msg("accessMode entered by user is " + accessMode)
	dbType := r.Form["dbtype"][0]

	m := authorization.AuthMap{
		Subject: u.UserEmail,
		Object:  authorization.ObjectPipeline,
		Action:  authorization.ActionAdmin,
	}

	// validate pipeline name
	re := regexp.MustCompile("^[a-z0-9]*$")
	if !re.MatchString(p.ObjectMeta.Name) {
		a := u.Copy("pipeline name not valid, needs to be only lowercase chars and numbers")
		a.ShowCreatePipeline(w, r)
		return
	}
	if !m.Authorized(u.DatabaseType) {
		a := u.Copy("user not authorized to create pipelines")
		a.ShowCreatePipeline(w, r)
		return
	}

	if dbPassword != dbPassword2 {
		a := u.Copy("database passwords do not match")
		a.ShowCreatePipeline(w, r)
		return
	}

	if p.ObjectMeta.Name == "" {
		a := u.Copy("pipeline name is blank")
		a.ShowCreatePipeline(w, r)
		return
	}

	i, err := strconv.Atoi(maxjobs)
	if i <= 0 {
		a := u.Copy("max jobs needs to be greater than 0")
		a.ShowCreatePipeline(w, r)
		return
	}

	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreatePipeline(w, r)
		return
	}

	// TODO validate storage size and class name here before trying

	p.Spec.MaxJobs = i
	p.Spec.DatabaseType = dbType
	p.Spec.StorageClassName = storageClassName
	p.Spec.StorageSize = storageSize
	p.Spec.AccessMode = accessMode
	p.Spec.AdminDataSource.Password = dbPassword
	p.Spec.DataSource.Password = dbPassword

	// we need to base64 encode the passwords
	sEnc := b64.StdEncoding.EncodeToString([]byte(dbPassword))
	p.Spec.AdminDataSource.Password = sEnc
	p.Spec.DataSource.Password = sEnc

	// create the CR to launch the pipeline on k8s
	log.Info().Msg("about to run pipeline.CreatePipeline with dbType " + p.Spec.DatabaseType)
	log.Info().Msg("about to run pipeline.CreatePipeline with accessMode " + p.Spec.AccessMode)
	err = pipeline.CreatePipeline(p)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreatePipeline(w, r)
		return
	}

	log.Info().Msg("created new pipeline..")
	http.Redirect(w, r, "/", 302)
}

// DeletePipelineDetail ...
func (u *HandlerWrapper) DeletePipelineDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]

	a := domain.Pipeline{}
	a.ID = pipelineID

	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionAdmin,
	}
	if !m.Authorized(u.DatabaseType) {
		w.Write([]byte("user not authorized to delete this pipeline"))
		return
	}

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	// delete the CR for this pipeline
	err = pipeline.DeletePipeline(x.Name)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	http.Redirect(w, r, "/", 302)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// PipelineDownloadFile ...
func (u *HandlerWrapper) PipelineDownloadFile(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	log.Info().Msg("pipeline download: id " + pipelineID)

	// get the CR, it holds all the credentials for this pipeline
	cr, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	// zip up the credentials

	buf := new(bytes.Buffer)

	// Create a new zip archive.
	zipWriter := zip.NewWriter(buf)

	// Add some files to the archive.
	var files = []struct {
		Name, Body string
	}{
		{"client.pipeline.key", cr.Spec.DatabaseCredentials.PipelineKey},
		{"client.pipeline.crt", cr.Spec.DatabaseCredentials.PipelineCrt},
		{"service.key", cr.Spec.ServiceCredentials.ServiceKey},
		{"service.crt", cr.Spec.ServiceCredentials.ServiceCrt},
		{"node.key", cr.Spec.DatabaseCredentials.NodeKey},
		{"node.crt", cr.Spec.DatabaseCredentials.NodeCrt},
		{"client.root.key", cr.Spec.DatabaseCredentials.ClientRootKey},
		{"client.root.crt", cr.Spec.DatabaseCredentials.ClientRootCrt},
		{"ca.key", cr.Spec.DatabaseCredentials.CAKey},
		{"ca.crt", cr.Spec.DatabaseCredentials.CACrt},
	}
	for _, file := range files {
		zipFile, err := zipWriter.Create(file.Name)
		if err != nil {
			a := u.Copy(err.Error())
			a.ShowCreatePipeline(w, r)
			return
		}
		_, err = zipFile.Write([]byte(file.Body))
		if err != nil {
			a := u.Copy(err.Error())
			a.ShowCreatePipeline(w, r)
			return
		}
	}

	err = zipWriter.Close()
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreatePipeline(w, r)
		return
	}

	w.Write(buf.Bytes())
}

// ShowCreatePipelineForm ...
type ShowCreatePipelineForm struct {
	ErrorText          string
	UserEmail          string
	StorageClasses     []string
	SupportedDatabases []string
}

// ShowCreatePipeline ...
func (u *HandlerWrapper) ShowCreatePipeline(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("pages/pipeline-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	names, err := GetStorageClasses()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	supportedDatabases, err := GetSupportedDatabases()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	x := ShowCreatePipelineForm{
		ErrorText:          u.ErrorText,
		UserEmail:          u.UserEmail,
		StorageClasses:     names,
		SupportedDatabases: supportedDatabases,
	}
	err = tmpl.ExecuteTemplate(w, "layout", x)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}

}

// DeletePipelines ...
func (u *HandlerWrapper) DeletePipelines(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

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

	pipelines := getSelectedPipelines(r.Form)
	log.Info().Msg(fmt.Sprintf("selected pipelines %+v", pipelines))

	for i := 0; i < len(pipelines); i++ {
		m := authorization.AuthMap{
			Subject:    u.UserEmail,
			PipelineID: pipelines[i],
			Object:     authorization.ObjectPipeline,
			Action:     authorization.ActionAdmin,
		}
		if !m.Authorized(u.DatabaseType) {
			w.Write([]byte("user not authorized to delete this pipeline"))
			return
		}

		var x v1alpha1.Pipeline
		for j := 0; j < len(pList.Items); j++ {
			if pipelines[i] == pList.Items[j].Spec.Id {
				x = pList.Items[j]
			}
		}

		// delete the CR for this pipeline
		err = pipeline.DeletePipeline(x.Name)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

	}

	http.Redirect(w, r, "/", 302)

}

func getSelectedPipelines(form url.Values) (values []string) {
	values = make([]string, 0)
	for k := range form {
		if strings.HasPrefix(k, "pipeline-") {
			pipelineID := strings.TrimPrefix(k, "pipeline-")
			values = append(values, pipelineID)
		}
	}

	return values
}
