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
	"time"

	"html/template"
	"net/http"

	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
)

// ProfilePage ...
type ProfilePage struct {
	UserEmail string
	Email     string
	Access    string
	FirstName string
	LastName  string
}

// CurrentTokens ...
var CurrentTokens AuthenticationTokens

// AuthenticationTokens ...
type AuthenticationTokens struct {
	// map with key of token and value of user email
	TokenUsers map[string]string
}

func init() {
	CurrentTokens = AuthenticationTokens{
		TokenUsers: make(map[string]string),
	}
}

// Logout ...
func (u *HandlerWrapper) Logout(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("logout called")

	c, err := r.Cookie("X-Session-Token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in cookie search")
		http.Redirect(w, r, "/login", 302)
		return
	}

	token := c.Value

	// delete the cached token
	delete(CurrentTokens.TokenUsers, token)

	// delete the stored cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "X-Session-Token",
		Value:  "",
		MaxAge: 0,
	})

	tmpl, err := template.ParseFiles("pages/logout.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pageValues := PipelinesPage{}
	err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// Login ...
func (u *HandlerWrapper) Login(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("login called " + time.Now().String())

	//tmpl, err := template.ParseFiles("pages/login.html", "pages/navbar.html")
	tmpl, err := template.ParseFiles("pages/login.html")
	if err != nil {
		log.Error().Err(err).Msg("error reading login html file")
		w.Write([]byte(err.Error()))
		return
	}

	pageValues := PipelinesPage{}
	err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// LoginTry ...
func (u *HandlerWrapper) LoginTry(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("logintry called")

	r.ParseForm()

	email := r.Form["email"][0]
	password := r.Form["pwd"][0]

	if email == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid email address")
		return
	}
	if password == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid password")
		return
	}

	log.Info().Msg(fmt.Sprintf("entered email %s password %s\n", email, password))

	churroDB, err := db.NewChurroDB(u.DatabaseType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	_, err = churroDB.Authenticate(email, password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "re-enter email/password")
		return
	}

	// add to list of current tokens
	token := xid.New().String()
	CurrentTokens.TokenUsers[token] = email
	//r.Header.Set("X-Session-Token", token)
	http.SetCookie(w, &http.Cookie{
		Name:    "X-Session-Token",
		Value:   token,
		Expires: time.Now().Add(820 * time.Second),
	})
	log.Info().Msg("setting cookie")

	u.UserEmail = email

	// on valid login, redirect to home page
	http.Redirect(w, r, "/", 302)
}

// ProfileUpdatePage ...
type ProfileUpdatePage struct {
	UserEmail string
	Access    string
	ErrorText string
	FirstName string
	LastName  string
	Password  string
}

// ProfileUpdate ...
func (u *HandlerWrapper) ProfileUpdate(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("ProfileUpdate called")

	r.ParseForm()

	us := domain.UserProfile{
		Email:     r.Form["email"][0],
		FirstName: r.Form["firstname"][0],
		LastName:  r.Form["lastname"][0],
		Access:    r.Form["access"][0],
		Password:  r.Form["pwd"][0],
	}

	pageValues := ProfileUpdatePage{
		UserEmail: u.UserEmail,
		Password:  "",
	}
	tmpl, err := template.ParseFiles("pages/profile.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
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
	if us.Email == "" {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "invalid email address"
		us, err = churroDB.GetUserProfileByEmail(u.UserEmail)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		pageValues.Access = us.Access
		pageValues.FirstName = us.FirstName
		pageValues.LastName = us.LastName

		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
		//respondWithError(w, http.StatusBadRequest, "Invalid email address")
		//return
	}
	if us.Password == "" {
		//respondWithError(w, http.StatusBadRequest, "Invalid password")
		//return
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = "invalid password"
		us, err = churroDB.GetUserProfileByEmail(u.UserEmail)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		pageValues.Access = us.Access
		pageValues.FirstName = us.FirstName
		pageValues.LastName = us.LastName

		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	err = churroDB.GetConnection(authorization.AdminDB.DBCreds, authorization.AdminDB.Source)
	if err != nil {
		//w.Write([]byte(err.Error()))
		//return
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = err.Error()
		us, err = churroDB.GetUserProfileByEmail(u.UserEmail)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		pageValues.Access = us.Access
		pageValues.FirstName = us.FirstName
		pageValues.LastName = us.LastName

		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	x, err := churroDB.GetUserProfileByEmail(us.Email)
	if err != nil {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = err.Error()
		us, err = churroDB.GetUserProfileByEmail(u.UserEmail)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		pageValues.Access = us.Access
		pageValues.FirstName = us.FirstName
		pageValues.LastName = us.LastName

		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	us.ID = x.ID
	err = churroDB.UpdateUserProfile(us)
	if err != nil {
		pageValues.UserEmail = u.UserEmail
		pageValues.ErrorText = err.Error()
		err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in template")
		}
		return
	}

	// on valid login, redirect to home page
	http.Redirect(w, r, "/", 302)
}

// Profile ...
func (u *HandlerWrapper) Profile(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("Profile called")

	tmpl, err := template.ParseFiles("pages/profile.html", "pages/navbar.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pageValues := ProfilePage{
		UserEmail: u.UserEmail,
		Email:     u.UserEmail,
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
	us, err := churroDB.GetUserProfileByEmail(u.UserEmail)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	pageValues.Access = us.Access
	pageValues.FirstName = us.FirstName
	pageValues.LastName = us.LastName

	err = tmpl.ExecuteTemplate(w, "layout", &pageValues)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in template")
	}
}

// Middleware function, which will be called for each request
func (amw *AuthenticationTokens) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info().Msg(r.RequestURI)
		if r.RequestURI == "/logout" || r.RequestURI == "/login" || r.RequestURI == "/logintry" {
			next.ServeHTTP(w, r)
			return
		}
		//token := r.Header.Get("X-Session-Token")
		c, err := r.Cookie("X-Session-Token")
		if err != nil {
			log.Error().Stack().Err(err).Msg("in cookie search")
			http.Redirect(w, r, "/login", 302)
			return
		}

		token := c.Value

		log.Info().Msg(fmt.Sprintf("middleware token found in header of [%s]\n", token))

		if user, found := CurrentTokens.TokenUsers[token]; found {
			// We found the token in our map
			log.Info().Msg("Authenticated user " + user)
			// Pass down the request to the next middleware (or final handler)
			next.ServeHTTP(w, r)
		} else {
			// Write an error and stop the handler chain
			//		http.Error(w, "Forbidden", http.StatusForbidden)
			http.Redirect(w, r, "/login", 302)
		}
	})
}
