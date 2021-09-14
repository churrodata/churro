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
	"time"

	"html/template"
	"net/http"

	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
)

// UserAdmin gets called when you add a new user
func (u *HandlerWrapper) UserAdmin(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("UserAdmin called")

	r.ParseForm()

	firstname := r.Form["firstname"][0]
	lastname := r.Form["lastname"][0]
	password := r.Form["password"][0]
	email := r.Form["email"][0]
	access := r.Form["access"][0]

	pageValues := NewUserDetailPage{
		UserEmail: u.UserEmail,
	}
	tmpl, err := template.ParseFiles("pages/user-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	if firstname == "" {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "first name invalid"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
		//respondWithError(w, http.StatusBadRequest, "Invalid username, first and last")
		//return
	}
	if lastname == "" {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "invalid username, first and last required"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if password == "" {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "invalid password"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if email == "" {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "invalid email"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if access == "" {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "invalid access"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	x := domain.UserProfile{
		FirstName:   firstname,
		LastName:    lastname,
		Password:    password,
		Email:       email,
		Access:      access,
		LastUpdated: time.Now(),
		ID:          xid.New().String()}

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)

	if err != nil {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)

		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.CreateUserProfile(x)
	if err != nil {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	log.Info().Msg("adding " + x.Email + " to UserProfiles")

	http.Redirect(w, r, "/users", 302)
}

// AccessValues ...
type AccessValues struct {
	Key      string
	Selected string
}

// UserDetailPage ...
type UserDetailPage struct {
	UserEmail string
	User      domain.UserProfile
	Values    []AccessValues
	ErrorText string
}

// NewUserDetailPage ...
type NewUserDetailPage struct {
	UserEmail string
	User      domain.UserProfile
	ErrorText string
}

// UserAdminDetail get a user's details
func (u *HandlerWrapper) UserAdminDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Info().Msg("UserAdminDetail called id=" + id)

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

	var up domain.UserProfile
	up, err = churroDB.GetUserProfile(id)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	dup := domain.UserProfile{
		ID:        id,
		FirstName: up.FirstName,
		LastName:  up.LastName,
		Password:  up.Password,
		Access:    up.Access,
		Email:     up.Email}

	tmpl, err := template.ParseFiles("pages/useradmindetail.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pageValues := UserDetailPage{
		User:      dup,
		UserEmail: u.UserEmail,
		Values:    make([]AccessValues, 0)}

	v := AccessValues{Key: "Read"}
	if dup.Access == "Read" {
		v.Selected = "selected"
	}
	pageValues.Values = append(pageValues.Values, v)
	v = AccessValues{Key: "Write"}
	if dup.Access == "Write" {
		v.Selected = "selected"
	}
	pageValues.Values = append(pageValues.Values, v)
	v = AccessValues{Key: "Admin"}
	if dup.Access == "Admin" {
		v.Selected = "selected"
	}
	pageValues.Values = append(pageValues.Values, v)

	err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// UserAdminUpdate ...
func (u *HandlerWrapper) UserAdminUpdate(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	r.ParseForm()

	firstname := r.Form["firstname"][0]
	lastname := r.Form["lastname"][0]
	password := r.Form["pwd"][0]
	email := r.Form["email"][0]
	access := r.Form["access"][0]

	// error handling
	tmpl, err := template.ParseFiles("pages/useradmindetail.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	up := domain.UserProfile{
		ID:        id,
		FirstName: firstname,
		LastName:  lastname,
		Password:  password,
		Access:    access,
		Email:     email}

	pageValues := UserDetailPage{
		User:      up,
		UserEmail: email,
		Values:    make([]AccessValues, 0)}
	v := AccessValues{Key: "Read"}
	if up.Access == "Read" {
		v.Selected = "selected"
	}
	pageValues.Values = append(pageValues.Values, v)
	v = AccessValues{Key: "Write"}
	if up.Access == "Write" {
		v.Selected = "selected"
	}
	pageValues.Values = append(pageValues.Values, v)
	v = AccessValues{Key: "Admin"}
	if up.Access == "Admin" {
		v.Selected = "selected"
	}
	pageValues.Values = append(pageValues.Values, v)
	// end of error handling setup

	if firstname == "" {
		pageValues.ErrorText = "first name invalid"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if lastname == "" {
		pageValues.ErrorText = "last name invalid"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if password == "" {
		pageValues.ErrorText = "password invalid"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if email == "" {
		pageValues.ErrorText = "email invalid"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}
	if access == "" {
		pageValues.ErrorText = "access invalid"
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	// get the UserProfile
	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	var existingUser domain.UserProfile
	existingUser, err = churroDB.GetUserProfile(id)
	if err != nil {
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	// update the UserProfile
	existingUser.Password = password
	existingUser.Access = access
	existingUser.Email = email
	existingUser.FirstName = firstname
	existingUser.LastName = lastname

	err = churroDB.UpdateUserProfile(existingUser)
	if err != nil {
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	http.Redirect(w, r, "/users", 302)
}

// ShowCreateUserForm ...
type ShowCreateUserForm struct {
	UserEmail string
	ErrorText string
}

// ShowCreateUser ...
func (u *HandlerWrapper) ShowCreateUser(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("pages/user-create.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	x := ShowCreateUserForm{
		ErrorText: u.ErrorText,
		UserEmail: u.UserEmail,
	}
	err = tmpl.ExecuteTemplate(w, "layout", &x)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}

}

// UsersPage ...
type UsersPage struct {
	UserEmail string
	List      []domain.UserProfile
}

// Users display all the users
func (u *HandlerWrapper) Users(w http.ResponseWriter, r *http.Request) {
	pageValues := UsersPage{
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

	pageValues.List, err = churroDB.GetAllUserProfile()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	tmpl, err := template.ParseFiles("pages/useradmin.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// UserAdminDelete ...
func (u *HandlerWrapper) UserAdminDelete(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]
	log.Info().Msg("UserAdminDelete called id=" + id)

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

	err = churroDB.DeleteUserProfile(id)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	http.Redirect(w, r, "/users", 302)
}
