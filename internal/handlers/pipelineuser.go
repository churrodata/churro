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
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// PipelineUsersForm ...
type PipelineUsersForm struct {
	PipelineName string
	PipelineID   string
	Users        []domain.UserProfile
	ErrorText    string
}

// PipelineUserForm ...
type PipelineUserForm struct {
	ErrorText    string
	PipelineID   string
	PipelineName string
	User         domain.UserProfile
}

// ShowPipelineUsers ...
func (u *HandlerWrapper) ShowPipelineUsers(w http.ResponseWriter, r *http.Request) {

	u.ErrorText = ""

	tmpl, err := template.ParseFiles("pages/pipeline-users.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	vars := mux.Vars(r)
	pipelineUsersForm := PipelineUsersForm{}
	pipelineUsersForm.PipelineID = vars["id"]

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		pipelineUsersForm.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		pipelineUsersForm.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	x, err := getPipelineCR(pipelineUsersForm.PipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pipelineUsersForm.PipelineName = x.Name

	var list []domain.UserProfile
	list, err = churroDB.GetAllUserProfile()
	if err != nil {
		pipelineUsersForm.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	pipelineUsersForm.Users = list

	//	pipelineUsersForm.ErrorText = u.ErrorText

	err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// UpdatePipelineUsers ...
func (u *HandlerWrapper) UpdatePipelineUsers(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	pipelineID := r.Form["pipelineid"][0]
	pipelineName := r.Form["pipelinename"][0]

	selectedUserIds := getSelectedUserIds(r.Form)

	// for error handling
	tmpl, err := template.ParseFiles("pages/pipeline-users.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	pipelineUsersForm := PipelineUsersForm{
		PipelineID:   pipelineID,
		PipelineName: pipelineName,
	}
	// end of error handling

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		pipelineUsersForm.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		pipelineUsersForm.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	//update the pipeline's list of users, remove all first, then
	// add this list since a user can select and deselect multiples

	// remove all first
	err = churroDB.DeleteAllUserPipelineAccess(pipelineID)
	if err != nil {
		pipelineUsersForm.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	// next, add in all the selected users
	//list := make([]domain.UserProfile, 0)
	for k, v := range selectedUserIds {
		userProf := domain.UserProfile{
			ID:     k,
			Access: v,
		}

		// get the UserProfile for this selection

		up, err := churroDB.GetUserProfile(k)
		if err != nil {
			pipelineUsersForm.ErrorText = err.Error()
			err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in template")
			}
			return
		}
		userProf.LastName = up.LastName
		userProf.FirstName = up.FirstName
		userProf.Email = up.Email

		uap := domain.UserPipelineAccess{
			PipelineID:    pipelineID,
			UserProfileID: k,
			Access:        v,
		}
		err = churroDB.CreateUserPipelineAccess(uap)
		if err != nil {
			pipelineUsersForm.ErrorText = err.Error()
			err = tmpl.ExecuteTemplate(w, "layout", pipelineUsersForm)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in template")
			}
			return
		}
	}

	targetURL := fmt.Sprintf("/pipelines/%s#nav-users", pipelineID)
	http.Redirect(w, r, targetURL, 302)
}

// DeletePipelineUser ...
func (u *HandlerWrapper) DeletePipelineUser(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	userID := vars["uid"]

	log.Info().Msg(fmt.Sprintf("deletepipelineuser id %s uid %s", pipelineID, userID))

	// for any errors we run into on this page
	tmpl, err := template.ParseFiles("pages/pipeline-user.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	form := PipelineUserForm{
		PipelineID:   pipelineID,
		PipelineName: "somepipeline",
	}

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	form.PipelineName = x.Name

	err = churroDB.DeleteUserPipelineAccess(pipelineID, userID)
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s", pipelineID)
	http.Redirect(w, r, targetURL, 302)

}

// return a map with the key being the user Id, and the value being
// the access value for that user Id
func getSelectedUserIds(form url.Values) (values map[string]string) {
	values = make(map[string]string)
	for k := range form {
		if strings.Contains(k, "remember-") {
			tmp := strings.Split(k, "-")
			userID := tmp[1]
			values[userID] = "not yet figured out"
		}
	}

	// for any userIDs selected, get the access value
	for k := range values {
		accessValue, found := form["access-"+k]
		if found {
			values[k] = accessValue[0]
		}
	}
	return values
}

// PipelineUser called when a pipeline user is viewed from a pipeline's list of users
func (u *HandlerWrapper) PipelineUser(w http.ResponseWriter, r *http.Request) {
	form := PipelineUserForm{}

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	userID := vars["userid"]

	form.PipelineID = pipelineID

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

	x, err := getPipelineCR(pipelineID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	form.PipelineName = x.Name

	// get the UserProfile for this user
	form.User, err = churroDB.GetUserProfile(userID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	// get the UserPipelineAccess for this pipeline/user
	var uap domain.UserPipelineAccess
	uap, err = churroDB.GetUserPipelineAccess(pipelineID, userID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	form.User.Access = uap.Access

	tmpl, err := template.ParseFiles("pages/pipeline-user.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	err = tmpl.ExecuteTemplate(w, "layout", form)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}

}

// UpdatePipelineUser ...
func (u *HandlerWrapper) UpdatePipelineUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uap := domain.UserPipelineAccess{}
	uap.UserProfileID = vars["uid"]

	r.ParseForm()
	uap.PipelineID = r.Form["pipelineid"][0]
	uap.Access = r.Form["useraccess"][0]
	form := PipelineUserForm{
		PipelineID: uap.PipelineID,
	}

	tmpl, err := template.ParseFiles("pages/pipeline-user.html", "pages/navbar.html")
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	err = churroDB.UpdateUserPipelineAccess(uap)
	if err != nil {
		form.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", form)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	targetURL := fmt.Sprintf("/pipelines/%s#nav-users", uap.PipelineID)
	http.Redirect(w, r, targetURL, 302)
}
