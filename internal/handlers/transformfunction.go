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
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// CreateTransformFunction ...
func (u *HandlerWrapper) CreateTransformFunction(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	pipelineID := r.Form["pipelineid"][0]
	pipelineName := r.Form["pipelinename"][0]

	p := domain.TransformFunction{
		ID:          xid.New().String(),
		Name:        r.Form["transformname"][0],
		Source:      r.Form["transformsource"][0],
		LastUpdated: time.Now(),
	}

	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionWrite,
	}
	if !m.Authorized(u.DatabaseType) {
		a := u.Copy("user not authorized to create transform function")
		a.ShowCreateTransformFunction(w, r)
		return
	}

	if p.Name == "" {
		a := u.Copy("transform name is blank")
		a.ShowCreateTransformFunction(w, r)
		return
	}
	if p.Source == "" {
		a := u.Copy("transform source is blank")
		a.ShowCreateTransformFunction(w, r)
		return
	}

	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	_, err := i.Eval(p.Source)
	if err != nil {
		a := u.Copy(err.Error())
		a.ErrorText = err.Error()
		a.ShowCreateTransformFunction(w, r)
		return

	}

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateTransformFunction(w, r)
		return
	}

	req := pb.CreateTransformFunctionRequest{}
	req.Namespace = pipelineName
	b, _ := json.Marshal(&p)
	req.FunctionString = string(b)

	_, err = client.CreateTransformFunction(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.ShowCreateTransformFunction(w, r)
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s#nav-transform-functions", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// FunctionFormValue ...
type FunctionFormValue struct {
	FunctionName string
	Selected     string
}

// TransformFunctionForm ...
type TransformFunctionForm struct {
	UserEmail    string
	PipelineName string
	PipelineID   string
	Functions    []FunctionFormValue
	ErrorText    string
}

// ShowCreateTransformFunction ...
func (u *HandlerWrapper) ShowCreateTransformFunction(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	log.Info().Msg("show-create-transform-function pipeline: id " + vars["id"])
	transformFunctionForm := TransformFunctionForm{
		PipelineID: vars["id"],
		UserEmail:  u.UserEmail,
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
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if transformFunctionForm.PipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	transformFunctionForm.PipelineName = x.Name

	tmpl, err := template.ParseFiles("pages/transform-function-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	transformFunctionForm.ErrorText = u.ErrorText

	err = tmpl.ExecuteTemplate(w, "layout", transformFunctionForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// FunctionForm ...
type FunctionForm struct {
	ErrorText    string
	UserEmail    string
	PipelineID   string
	PipelineName string
	Function     domain.TransformFunction
}

// TransformFunction ...
func (u *HandlerWrapper) TransformFunction(w http.ResponseWriter, r *http.Request) {

	ff := FunctionForm{}
	ff.UserEmail = u.UserEmail
	vars := mux.Vars(r)
	pipelineID := vars["id"]
	functionID := vars["tfid"]

	ff.PipelineID = pipelineID

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
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	ff.PipelineName = x.Name
	pipelineName := x.Name

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineDetailHandler(w, r)
		return
	}

	req := pb.GetTransformFunctionRequest{
		Namespace:  pipelineName,
		FunctionID: functionID}

	response, err := client.GetTransformFunction(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineDetailHandler(w, r)
		return
	}

	var function domain.TransformFunction
	err = json.Unmarshal([]byte(response.FunctionString), &function)
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineDetailHandler(w, r)
		return
	}
	ff.Function = function

	tmpl, err := template.ParseFiles("pages/transform-function.html", "pages/navbar.html")
	if err != nil {
		a := u.Copy(err.Error())
		a.PipelineDetailHandler(w, r)
		return
	}
	ff.ErrorText = u.ErrorText
	err = tmpl.ExecuteTemplate(w, "layout", ff)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// UpdateTransformFunction ...
func (u *HandlerWrapper) UpdateTransformFunction(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	f := domain.TransformFunction{Name: r.Form["transformname"][0]}
	pipelineID := r.Form["pipelineid"][0]
	pipelineName := r.Form["pipelinename"][0]
	if f.Name == "" {
		a := u.Copy("transform name can not be blank")
		a.TransformFunction(w, r)
		return
	}
	f.Source = r.Form["transformsource"][0]
	if f.Source == "" {
		a := u.Copy("transform Source can not be blank")
		a.TransformFunction(w, r)
		return
	}

	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	_, err := i.Eval(f.Source)
	if err != nil {
		a := u.Copy(err.Error())
		a.ErrorText = err.Error()
		a.TransformFunction(w, r)
		return
	}

	f.ID = r.Form["functionid"][0]

	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionWrite,
	}
	if !m.Authorized(u.DatabaseType) {
		a := u.Copy("user not authorized to update this transform function")
		a.TransformFunction(w, r)
		return
	}

	client, err := GetServiceConnection(pipelineName)
	if err != nil {
		a := u.Copy(err.Error())
		a.TransformFunction(w, r)
		return
	}

	b, _ := json.Marshal(&f)
	req := pb.UpdateTransformFunctionRequest{Namespace: pipelineName,
		FunctionString: string(b)}

	_, err = client.UpdateTransformFunction(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.TransformFunction(w, r)
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s#nav-transform-functions", pipelineID)
	http.Redirect(w, r, targetURL, 302)
}

// DeleteTransformFunction ...
func (u *HandlerWrapper) DeleteTransformFunction(w http.ResponseWriter, r *http.Request) {

	// TransformFunctions
	vars := mux.Vars(r)
	pipelineID := vars["id"]
	functionID := vars["tfid"]

	log.Info().Msg(fmt.Sprintf("delete transformfunction method called vars %+v\n", vars))

	m := authorization.AuthMap{
		Subject:    u.UserEmail,
		PipelineID: pipelineID,
		Object:     authorization.ObjectPipeline,
		Action:     authorization.ActionWrite,
	}
	if !m.Authorized(u.DatabaseType) {
		a := u.Copy("user not authorized to delete this transform function")
		a.TransformFunction(w, r)
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
		a := u.Copy(err.Error())
		a.TransformFunction(w, r)
		return
	}

	req := pb.DeleteTransformFunctionRequest{
		Namespace:  x.Name,
		FunctionID: functionID,
	}

	_, err = client.DeleteTransformFunction(context.Background(), &req)
	if err != nil {
		a := u.Copy(err.Error())
		a.TransformFunction(w, r)
		return
	}
	url := fmt.Sprintf("/pipelines/%s#nav-transform-functions", pipelineID)
	http.Redirect(w, r, url, 302)

}

func getTransformFunctions(pipelineName string) (list []domain.TransformFunction, err error) {
	var client pb.CtlClient
	client, err = GetServiceConnection(pipelineName)
	if err != nil {
		return list, err
	}
	req := pb.GetTransformFunctionsRequest{Namespace: pipelineName}

	response, err := client.GetTransformFunctions(context.Background(), &req)
	if err != nil {
		return list, err
	}
	byt := []byte(response.FunctionsString)
	err = json.Unmarshal(byt, &list)
	if err != nil {
		return list, err
	}
	return list, nil
}
