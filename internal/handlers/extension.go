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
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"

	"net/http"
	"time"

	"github.com/churrodata/churro/internal/domain"
	pb "github.com/churrodata/churro/rpc/ctl"
)

// ExtensionForm ...
type ExtensionForm struct {
	UserEmail         string
	ErrorText         string
	PipelineID        string
	PipelineName      string
	ExtractSourceID   string
	ExtractSourceName string
	ExtensionID       string
	ExtensionName     string
	ExtensionPath     string
}

// UpdateExtension ...
func (u *HandlerWrapper) UpdateExtension(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	if pipelineID == "" {
		a := u.Copy("invalid pipeline id")
		a.ShowCreateExtension(w, r)
		return
	}
	extractSourceID := vars["extractsourceid"]
	if extractSourceID == "" {
		a := u.Copy("invalid extract source id")
		a.ShowCreateExtension(w, r)
		return
	}
	extensionID := vars["eid"]
	if extensionID == "" {
		a := u.Copy("invalid extension id")
		a.ShowCreateExtension(w, r)
		return
	}

	r.ParseForm()

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	client, err := GetServiceConnection(x.Name)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	//  update the extension with the form contents
	ext := domain.Extension{
		ID:              extensionID,
		ExtractSourceID: extractSourceID,
		ExtensionName:   r.Form["extensionname"][0],
		ExtensionPath:   r.Form["extensionpath"][0],
	}

	req := pb.UpdateExtensionRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
	}

	b, _ := json.Marshal(&ext)
	req.ExtensionString = string(b)

	_, err = client.UpdateExtension(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		a := u.Copy(err.Error())
		a.Extension(w, r)
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// DeleteExtension ...
func (u *HandlerWrapper) DeleteExtension(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	req := pb.DeleteExtensionRequest{
		ExtractSourceID: vars["extractsourceid"],
		ExtensionID:     vars["eid"],
	}

	pipelineID := vars["id"]
	log.Info().Msg(fmt.Sprintf("ui DeleteExtension with extractsourceid=[%s] eid=[%s] id=[%s] ns=[%s]\n", req.ExtractSourceID, req.ExtensionID, pipelineID, req.Namespace))

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	client, err := GetServiceConnection(x.Name)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error DeleteExtension "+err.Error())
		return
	}

	req.Namespace = x.Name

	_, err = client.DeleteExtension(context.Background(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error DeleteExtension "+err.Error())
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s/extractsources/%s",
		pipelineID, req.ExtractSourceID)
	http.Redirect(w, r, targetURL, 302)
}

// ShowCreateExtension ...
func (u *HandlerWrapper) ShowCreateExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	extForm := ExtensionForm{
		UserEmail:       u.UserEmail,
		PipelineID:      vars["id"],
		ExtractSourceID: vars["extractsourceid"],
	}

	x, err := getPipelineCR(extForm.PipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	extForm.PipelineName = x.Name

	client, err := GetServiceConnection(extForm.PipelineName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.GetExtractSourceRequest{
		Namespace:       extForm.PipelineName,
		ExtractSourceID: vars["extractsourceid"],
	}

	response, err := client.GetExtractSource(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	var extractSource domain.ExtractSource
	err = json.Unmarshal([]byte(response.ExtractSourceString), &extractSource)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	extForm.ExtractSourceName = extractSource.Name

	tmpl, err := template.ParseFiles("pages/extension-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	extForm.ErrorText = u.ErrorText
	err = tmpl.ExecuteTemplate(w, "layout", extForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}

}

// CreateExtension ...
func (u *HandlerWrapper) CreateExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]

	r.ParseForm()

	p := domain.Extension{
		ID:              xid.New().String(),
		ExtractSourceID: extractSourceID,
		ExtensionName:   r.Form["extensionname"][0],
		ExtensionPath:   r.Form["extensionpath"][0],
		LastUpdated:     time.Now(),
	}
	pipelineName := r.Form["pipelinename"][0]

	if p.ExtensionName == "" {
		a := u.Copy("extension name is blank")
		a.ShowCreateExtension(w, r)
		return
	}
	if p.ExtensionPath == "" {
		a := u.Copy("extension path is blank")
		a.ShowCreateExtension(w, r)
		return
	}

	client, err := GetServiceConnection(pipelineName)
	if err != nil {

		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	b, _ := json.Marshal(&p)
	req := pb.CreateExtensionRequest{
		Namespace:       x.Name,
		ExtensionString: string(b),
	}

	log.Info().Msg("CreateExtension being called here")
	_, err = client.CreateExtension(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}
	targetURL := fmt.Sprintf("/pipelines/%s/extractsources/%s?pipelinename=%s", pipelineID, extractSourceID, pipelineName)
	http.Redirect(w, r, targetURL, 302)

}

// Extension ...
func (u *HandlerWrapper) Extension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Info().Msg("Extension called...")
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]
	extensionID := vars["eid"]

	log.Info().Msg("pipeline: id " + pipelineID)
	log.Info().Msg("pipeline: extractsourceid " + extractSourceID)
	log.Info().Msg("pipeline: eid " + extensionID)

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pipelineName := x.Name

	log.Info().Msg("pipeline: pipelinename " + pipelineName)

	client, err := GetServiceConnection(x.Name)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	wreq := pb.GetExtractSourceRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
	}
	wresponse, err := client.GetExtractSource(context.Background(), &wreq)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in getextractsource")
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	var extractSource domain.ExtractSource
	err = json.Unmarshal([]byte(wresponse.ExtractSourceString), &extractSource)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	log.Info().Msg("pipeline: extract source name " + extractSource.Name)

	req := pb.GetExtensionRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
		ExtensionID:     extensionID,
	}

	response, err := client.GetExtension(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in getextension ")
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	var ext domain.Extension
	err = json.Unmarshal([]byte(response.ExtensionString), &ext)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in extension unmarshal")
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}

	extForm := ExtensionForm{
		UserEmail:         u.UserEmail,
		PipelineID:        pipelineID,
		PipelineName:      pipelineName,
		ExtractSourceID:   extractSourceID,
		ExtractSourceName: extractSource.Name,
		ExtensionID:       extensionID,
		ExtensionName:     ext.ExtensionName,
		ExtensionPath:     ext.ExtensionPath,
	}

	tmpl, err := template.ParseFiles("pages/extension.html", "pages/navbar.html")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
		a := u.Copy(err.Error())
		a.ShowCreateExtension(w, r)
		return
	}
	extForm.ErrorText = u.ErrorText
	err = tmpl.ExecuteTemplate(w, "layout", extForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}
