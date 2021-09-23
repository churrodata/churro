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
	"bufio"
	"context"
	"encoding/json"
	"strconv"

	"github.com/rs/zerolog/log"

	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/extractsource"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	watchpb "github.com/churrodata/churro/rpc/extractsource"
	"github.com/gorilla/mux"
	"github.com/rs/xid"
)

// ExtractSourceForm ...
type ExtractSourceForm struct {
	Initialized     bool
	Running         bool
	UserEmail       string
	ErrorText       string
	StatusText      string
	PipelineID      string
	PipelineName    string
	ExtractSourceID string
	ExtractSource   domain.ExtractSource
	ExtractRules    []domain.ExtractRule
	Extensions      []domain.Extension
	Metrics         []domain.ExtractSourceMetric
}

// ShowCreateExtractSource ...
func (u *HandlerWrapper) ShowCreateExtractSource(w http.ResponseWriter, r *http.Request) {

	//u.ErrorText = ""
	vars := mux.Vars(r)
	log.Info().Msg(fmt.Sprintf("ShowCreateExtractSource called : vars %+v\n", vars))
	pipelineID := vars["id"]

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
		//return
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	extractSourceForm := ExtractSourceForm{
		PipelineID:      pipelineID,
		PipelineName:    x.Name,
		ErrorText:       u.ErrorText,
		UserEmail:       u.UserEmail,
		ExtractSourceID: vars["extractsourceid"]}

	tmpl, err := template.ParseFiles("pages/extractsource-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout", extractSourceForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// PipelineExtractSource ...
func (u *HandlerWrapper) PipelineExtractSource(w http.ResponseWriter, r *http.Request) {

	wdf := ExtractSourceForm{}

	vars := mux.Vars(r)
	wdf.PipelineID = vars["id"]
	wdf.ExtractSourceID = vars["extractsourceid"]

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
		if wdf.PipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	client, err := GetServiceConnection(x.Name)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.GetExtractSourceRequest{
		Namespace:       x.Name,
		ExtractSourceID: wdf.ExtractSourceID,
	}

	response, err := client.GetExtractSource(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	var value domain.ExtractSource
	err = json.Unmarshal([]byte(response.ExtractSourceString), &value)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	wdf.ExtractSource = value
	log.Info().Msg(fmt.Sprintf("wdir.ExtractSource is %v\n", value))
	wdf.PipelineName = x.Name
	wdf.UserEmail = u.UserEmail

	wdf.Metrics = make([]domain.ExtractSourceMetric, 0)
	for i := 0; i < len(response.Metrics); i++ {
		mt := domain.ExtractSourceMetric{}
		mt.Name = response.Metrics[i].GetName()
		mt.Value = response.Metrics[i].GetValue()
		wdf.Metrics = append(wdf.Metrics, mt)
	}

	for _, v := range value.ExtractRules {
		wdf.ExtractRules = append(wdf.ExtractRules, v)
	}
	for _, v := range value.Extensions {
		wdf.Extensions = append(wdf.Extensions, v)
	}

	tmpl, err := template.ParseFiles("pages/extractsource.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	wdf.ErrorText = u.ErrorText
	wdf.StatusText = u.StatusText
	err = tmpl.ExecuteTemplate(w, "layout", wdf)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// CreateExtractSource ...
func (u *HandlerWrapper) CreateExtractSource(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	pipelineID := r.Form["pipelineid"][0]

	//validate the skipheaders as an integer
	v, err := strconv.Atoi(r.Form["skipheaders"][0])
	if err != nil {
		a := u.Copy("skipheaders is blank or not a valid integer")
		a.ShowCreateExtractSource(w, r)
		return
	}

	rawValue := r.Form["multiline"][0]
	if rawValue == "" {
		rawValue = "false"
	}
	mV, err := strconv.ParseBool(rawValue)
	if err != nil {
		a := u.Copy("multiline is blank or not a valid boolean")
		a.ShowCreateExtractSource(w, r)
		return
	}

	p, err := strconv.Atoi(r.Form["port"][0])
	if err != nil {
		a := u.Copy("port is blank or not a valid integer")
		a.ShowCreateExtractSource(w, r)
		return
	}

	d := domain.ExtractSource{
		ID:             xid.New().String(),
		Name:           r.Form["extractsourcename"][0],
		Path:           r.Form["extractsourcepath"][0],
		Scheme:         r.Form["extractsourcescheme"][0],
		Regex:          r.Form["extractsourceregex"][0],
		Tablename:      r.Form["extractsourcetablename"][0],
		Cronexpression: r.Form["cronexpression"][0],
		Multiline:      mV,
		Sheetname:      r.Form["sheetname"][0],
		Skipheaders:    v,
		Port:           p,
		Encoding:       r.Form["encoding"][0],
		Transport:      r.Form["transport"][0],
		Servicetype:    r.Form["servicetype"][0],
		LastUpdated:    time.Now(),
		ExtractRules:   make(map[string]domain.ExtractRule),
	}
	pipelineName := r.Form["pipelinename"][0]

	if d.Path == "" {
		a := u.Copy("Path is blank")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Scheme == "" {
		a := u.Copy("scheme is blank")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Tablename == "" {
		a := u.Copy("tablename is blank")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Regex == "" && d.Scheme != extractapi.APIScheme {
		a := u.Copy("regex is blank")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Name == "" {
		a := u.Copy("name is blank")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Skipheaders < 0 && (d.Scheme == extractapi.CSVScheme || d.Scheme == extractapi.XLSXScheme) {
		a := u.Copy("skipheaders is required to be >= 0")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Servicetype == "" && (d.Scheme == extractapi.HTTPPostScheme) {
		a := u.Copy("servicetype is required")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Transport == "" && (d.Scheme == extractapi.HTTPPostScheme) {
		a := u.Copy("transport is required")
		a.ShowCreateExtractSource(w, r)
		return
	}
	if d.Sheetname == "" && (d.Scheme == extractapi.XLSXScheme) {
		a := u.Copy("sheetname is required to be non-blank")
		a.ShowCreateExtractSource(w, r)
		return
	}

	log.Info().Msg(fmt.Sprintf("adding new extract source %+v\n", d))

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractSource(w, r)
		return
	}

	b, _ := json.Marshal(&d)
	req := pb.CreateExtractSourceRequest{
		Namespace:           pipelineName,
		ExtractSourceString: string(b),
	}

	_, err = client.CreateExtractSource(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in creating extract source ")
		a := u.Copy(err.Error())
		a.ShowCreateExtractSource(w, r)
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s#nav-extractsources", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// UpdateExtractSource ...
func (u *HandlerWrapper) UpdateExtractSource(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	pipelineName := r.Form["pipelinename"][0]
	extractSourceID := r.Form["extractsourceid"][0]

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	req := pb.GetExtractSourceRequest{
		Namespace:       pipelineName,
		ExtractSourceID: extractSourceID}

	response, err := client.GetExtractSource(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	var wdir domain.ExtractSource
	err = json.Unmarshal([]byte(response.ExtractSourceString), &wdir)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	wdir.Name = r.Form["name"][0]
	if wdir.Name == "" {
		a := u.Copy("name is blank")
		a.PipelineExtractSource(w, r)
		return
	}
	wdir.Path = r.Form["path"][0]
	if wdir.Path == "" {
		a := u.Copy("path is blank")
		a.PipelineExtractSource(w, r)
		return
	}
	wdir.Regex = r.Form["regex"][0]
	if wdir.Regex == "" && wdir.Scheme != extractapi.APIScheme {
		a := u.Copy("regex is blank")
		a.PipelineExtractSource(w, r)
		return
	}
	wdir.Scheme = r.Form["scheme"][0]
	if wdir.Scheme == "" {
		a := u.Copy("scheme is blank")
		a.PipelineExtractSource(w, r)
		return
	}
	wdir.Cronexpression = r.Form["cronexpression"][0]
	if wdir.Cronexpression == "" && wdir.Scheme == extractapi.APIScheme {
		a := u.Copy("cronexpression is blank")
		a.PipelineExtractSource(w, r)
		return
	}
	wdir.Tablename = r.Form["tablename"][0]
	if wdir.Tablename == "" {
		a := u.Copy("tablename is blank")
		a.PipelineExtractSource(w, r)
		return
	}

	b, _ := json.Marshal(&wdir)
	wreq := pb.UpdateExtractSourceRequest{
		Namespace:           pipelineName,
		ExtractSourceString: string(b),
	}

	_, err = client.UpdateExtractSource(context.Background(), &wreq)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}
	targetURL := fmt.Sprintf("/pipelines/%s#nav-extractsources", r.Form["pipelineid"][0])
	http.Redirect(w, r, targetURL, 302)
}

// DeleteExtractSource ...
func (u *HandlerWrapper) DeleteExtractSource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	log.Info().Msg(fmt.Sprintf("DeleteExtractSource called...vars %+v\n", vars))
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]

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
		//return
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	client, err := GetServiceConnection(x.Name)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.DeleteExtractSourceRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
	}

	_, err = client.DeleteExtractSource(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	targetURL := fmt.Sprintf("/pipelines/%s#nav-extractsources", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// UploadURLToExtractSource ...
func (u *HandlerWrapper) UploadURLToExtractSource(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Info().Msg(fmt.Sprintf("UploadURLToExtractSource called form %+v\n", r.Form))
	extractSourceID := r.Form["extractsourceid"][0]
	pipelineID := r.Form["pipelineid"][0]
	fileURL := r.Form["fileurl"][0]

	req := watchpb.UploadByURLRequest{
		ExtractSourceID: extractSourceID,
		PipelineID:      pipelineID,
		FileURL:         fileURL,
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
		//return
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	client, err := extractsource.GetExtractSourceServiceConnection(x.Name)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	response, err := client.UploadByURL(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg(fmt.Sprintf("%+v", response))
		w.Write([]byte(err.Error()))
		return
	}

	j := HandlerWrapper{}
	j.ErrorText = ""
	j.DatabaseType = u.DatabaseType
	j.StatusText = "URL uploaded successfully"
	j.PipelineExtractSource(w, r)
	return
}

// UploadToExtractSource ...
func (u *HandlerWrapper) UploadToExtractSource(w http.ResponseWriter, r *http.Request) {

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Error Retrieving the File")
		w.Write([]byte(err.Error()))
		return
	}
	defer file.Close()

	log.Info().Msg(fmt.Sprintf("Uploaded File: %+v\n", handler.Filename))
	log.Info().Msg(fmt.Sprintf("File Size: %+v\n", handler.Size))
	log.Info().Msg(fmt.Sprintf("MIME Header: %+v\n", handler.Header))

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]

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

	client, err := extractsource.GetExtractSourceServiceConnection(x.Name)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.UploadToExtractSource(ctx)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	log.Info().Msg("uploading with extractSourceID " + extractSourceID)
	req := watchpb.UploadToExtractSourceRequest{
		Data: &watchpb.UploadToExtractSourceRequest_Info{
			Info: &watchpb.UploadInfo{
				Namespace:       x.Name,
				ExtractSourceID: extractSourceID,
				FileType:        "csv",
				FileName:        handler.Filename,
			},
		},
	}
	err = stream.Send(&req)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			a := u.Copy(err.Error())
			a.PipelineExtractSource(w, r)
			return
		}

		req := &watchpb.UploadToExtractSourceRequest{
			Data: &watchpb.UploadToExtractSourceRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}
		err = stream.Send(req)
		if err != nil {
			a := u.Copy(err.Error())
			a.PipelineExtractSource(w, r)
			return
		}
	}

	response, err := stream.CloseAndRecv()
	log.Info().Msg(fmt.Sprintf("stream response %+v\n", response))
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineExtractSource(w, r)
		return
	}

	log.Info().Msg(fmt.Sprintf("upload response %+v\n", response))

	j := HandlerWrapper{}
	j.ErrorText = ""
	j.DatabaseType = u.DatabaseType
	j.StatusText = "file uploaded successfully"
	j.PipelineExtractSource(w, r)
	return
}
