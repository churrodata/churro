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

// ExtractRuleForm ...
type ExtractRuleForm struct {
	UserEmail         string
	ErrorText         string
	PipelineID        string
	PipelineName      string
	ExtractSourceID   string
	ExtractSourceName string
	ExtractRuleID     string
	ColumnName        string
	ColumnPath        string
	ColumnType        string
	MatchValues       string
	Initialized       bool
	Functions         []FunctionFormValue
}

// UpdateExtractRule ...
func (u *HandlerWrapper) UpdateExtractRule(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	if pipelineID == "" {
		a := u.Copy("invalid pipeline id")
		a.ShowCreateExtractRule(w, r)
		return
	}
	extractSourceID := vars["extractsourceid"]
	if extractSourceID == "" {
		a := u.Copy("invalid extract source id")
		a.ShowCreateExtractRule(w, r)
		return
	}
	ruleID := vars["rid"]
	if ruleID == "" {
		a := u.Copy("invalid rule id")
		a.ShowCreateExtractRule(w, r)
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
		a.ShowCreateExtractRule(w, r)
		return
	}

	//  update the rule with the form contents
	rule := domain.ExtractRule{
		ID:              ruleID,
		ExtractSourceID: extractSourceID,
		ColumnName:      r.Form["columnname"][0],
		ColumnPath:      r.Form["columnpath"][0],
		ColumnType:      r.Form["columntype"][0],
		MatchValues:     r.Form["matchvalues"][0],
	}

	req := pb.UpdateExtractRuleRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
	}

	b, _ := json.Marshal(&rule)
	req.ExtractRuleString = string(b)

	_, err = client.UpdateExtractRule(context.Background(), &req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		a := u.Copy(err.Error())
		//a.ShowCreateExtractRule(w, r)
		a.ExtractRule(w, r)
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// DeleteExtractRule ...
func (u *HandlerWrapper) DeleteExtractRule(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	req := pb.DeleteExtractRuleRequest{
		ExtractSourceID: vars["extractsourceid"],
		ExtractRuleID:   vars["rid"],
	}

	pipelineID := vars["id"]

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req.Namespace = x.Name
	client, err := GetServiceConnection(req.Namespace)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error DeleteExtractRule "+err.Error())
		return
	}

	_, err = client.DeleteExtractRule(context.Background(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error DeleteExtractRule "+err.Error())
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s/extractsources/%s",
		pipelineID, req.ExtractSourceID)
	http.Redirect(w, r, targetURL, 302)
}

// ShowCreateExtractRule ...
func (u *HandlerWrapper) ShowCreateExtractRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	extractRuleForm := ExtractRuleForm{
		UserEmail:       u.UserEmail,
		PipelineID:      vars["id"],
		ExtractSourceID: vars["extractsourceid"],
		Functions:       make([]FunctionFormValue, 0),
	}

	// allow extract rules to not specify a transform function
	f := FunctionFormValue{
		FunctionName: "None",
		Selected:     "",
	}

	x, err := getPipelineCR(extractRuleForm.PipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	extractRuleForm.PipelineName = x.Name

	functions, err := getTransformFunctions(extractRuleForm.PipelineName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	client, err := GetServiceConnection(extractRuleForm.PipelineName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.GetExtractSourceRequest{
		Namespace:       extractRuleForm.PipelineName,
		ExtractSourceID: vars["extractsourceid"],
	}

	response, err := client.GetExtractSource(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	//response.ExtractRuleString
	var extractSource domain.ExtractSource
	err = json.Unmarshal([]byte(response.ExtractSourceString), &extractSource)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	extractRuleForm.ExtractSourceName = extractSource.Name

	extractRuleForm.Functions = append(extractRuleForm.Functions, f)

	for i := 0; i < len(functions); i++ {
		f := FunctionFormValue{
			FunctionName: functions[i].Name,
			Selected:     "",
		}
		extractRuleForm.Functions = append(extractRuleForm.Functions, f)
	}

	tmpl, err := template.ParseFiles("pages/extract-rules-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	extractRuleForm.ErrorText = u.ErrorText
	err = tmpl.ExecuteTemplate(w, "layout", extractRuleForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}

}

// CreateExtractRule ...
func (u *HandlerWrapper) CreateExtractRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]

	r.ParseForm()

	p := domain.ExtractRule{
		ID:                xid.New().String(),
		ExtractSourceID:   extractSourceID,
		ColumnName:        r.Form["columnname"][0],
		ColumnPath:        r.Form["columnpath"][0],
		ColumnType:        r.Form["columntype"][0],
		MatchValues:       r.Form["matchvalues"][0],
		TransformFunction: r.Form["transformfunctionname"][0],
		LastUpdated:       time.Now(),
	}
	pipelineName := r.Form["pipelinename"][0]

	if p.ColumnName == "" {
		a := u.Copy("column name is blank")
		a.ShowCreateExtractRule(w, r)
		return
	}
	if p.ColumnPath == "" {
		a := u.Copy("rule source is blank")
		a.ShowCreateExtractRule(w, r)
		return
	}

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}

	b, _ := json.Marshal(&p)
	req := pb.CreateExtractRuleRequest{
		Namespace:         x.Name,
		ExtractRuleString: string(b),
	}

	_, err = client.CreateExtractRule(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}
	targetURL := fmt.Sprintf("/pipelines/%s/extractsources/%s?pipelinename=%s", pipelineID, extractSourceID, pipelineName)
	http.Redirect(w, r, targetURL, 302)

}

// ExtractRule ...
func (u *HandlerWrapper) ExtractRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Info().Msg("ExtractRule called...")
	pipelineID := vars["id"]
	extractSourceID := vars["extractsourceid"]
	ruleID := vars["rid"]

	log.Info().Msg("pipeline: id " + pipelineID)
	log.Info().Msg("pipeline: extractsourceid " + extractSourceID)
	log.Info().Msg("pipeline: rid " + ruleID)

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
		a.ShowCreateExtractRule(w, r)
		return
	}

	wreq := pb.GetExtractSourceRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
	}

	wresponse, err := client.GetExtractSource(context.Background(), &wreq)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}

	var extractSource domain.ExtractSource
	err = json.Unmarshal([]byte(wresponse.ExtractSourceString), &extractSource)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}

	log.Info().Msg("pipeline: extract source name " + extractSource.Name)

	req := pb.GetExtractRuleRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
		ExtractRuleID:   ruleID,
	}

	response, err := client.GetExtractRule(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}

	var rule domain.ExtractRule
	err = json.Unmarshal([]byte(response.ExtractRuleString), &rule)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}

	extractRuleForm := ExtractRuleForm{
		UserEmail:         u.UserEmail,
		PipelineID:        pipelineID,
		PipelineName:      pipelineName,
		ExtractSourceID:   extractSourceID,
		ExtractSourceName: extractSource.Name,
		ExtractRuleID:     ruleID,
		ColumnName:        rule.ColumnName,
		MatchValues:       rule.MatchValues,
		ColumnPath:        rule.ColumnPath,
		ColumnType:        rule.ColumnType,
		Initialized:       extractSource.Initialized,
	}

	var functions []domain.TransformFunction
	functions, err = getTransformFunctions(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}

	noneValue := domain.TransformFunction{Name: ""}
	functions = append(functions, noneValue)

	extractRuleForm.Functions = make([]FunctionFormValue, 0)
	for i := 0; i < len(functions); i++ {
		x := FunctionFormValue{
			FunctionName: functions[i].Name,
		}
		if rule.TransformFunction == functions[i].Name {
			x.Selected = "selected"
		} else {
			x.Selected = ""
		}
		extractRuleForm.Functions = append(extractRuleForm.Functions, x)
	}

	tmpl, err := template.ParseFiles("pages/extract-rule.html", "pages/navbar.html")
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateExtractRule(w, r)
		return
	}
	extractRuleForm.ErrorText = u.ErrorText
	err = tmpl.ExecuteTemplate(w, "layout", extractRuleForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}
