// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package singlestore

import (
	"database/sql"
	"fmt"

	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
)

func (d SinglestoreChurroDatabase) CreateAuthenticatedUser(u domain.AuthenticatedUser) error {
	u.ID = xid.New().String()
	INSERT := "INSERT INTO authenticateduser(id, token, locked, lastupdated) values(?,?,?,now())"

	stmt, err := d.Connection.Prepare(INSERT)
	if err != nil {
		log.Error().Stack().Err(err).Msg(INSERT)
		return err
	}

	_, err = stmt.Exec(u.ID, u.Token, u.Locked)
	if err != nil {
		log.Error().Stack().Err(err)
	}

	return err
}

func (d SinglestoreChurroDatabase) CreateChurroDatabase(dbName string) error {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err := d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err).Msg(sqlStr)
		return err
	}
	log.Info().Msg("Successfully created database " + dbName)
	return nil
}

func (d SinglestoreChurroDatabase) CreateAuthObjects() error {
	// create AuthenticatedUser
	_, err := d.Connection.Exec("CREATE TABLE if not exists authenticateduser (id VARCHAR(255) PRIMARY KEY, token VARCHAR(64) NOT NULL, locked boolean NOT NULL, lastupdated TIMESTAMP NULL)")
	if err != nil {
		return err
	}

	// create UserProfile
	_, err = d.Connection.Exec("CREATE TABLE if not exists userprofile (id VARCHAR(255) PRIMARY KEY, firstname VARCHAR(64) NOT NULL, lastname VARCHAR(64) NOT NULL, password VARCHAR(64) NOT NULL, access VARCHAR(25) NOT NULL, email VARCHAR(64) NOT NULL, lastupdated TIMESTAMP NULL)")
	if err != nil {
		return err
	}
	// create UserPipelineAccess
	_, err = d.Connection.Exec("CREATE TABLE if not exists userpipelineaccess (userprofileid VARCHAR(255) NOT NULL, pipelineid VARCHAR(64) NOT NULL, access VARCHAR(25) NOT NULL, lastupdated TIMESTAMP NULL)")
	if err != nil {
		return err
	}
	// create Pipeline
	/**
	_, err = d.Connection.Exec("CREATE TABLE if not exists pipeline (id VARCHAR(255) PRIMARY KEY, name VARCHAR(64) NOT NULL, lastupdated TIMESTAMP NULL)")
	if err != nil {
		return err
	}
	*/

	return nil
}

func (d SinglestoreChurroDatabase) DeleteAuthenticatedUser(id string) (err error) {
	_, err = d.Connection.Exec(fmt.Sprintf("DELETE FROM authenticateduser where id='%s'", id))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) GetUserPipelineAccess(pipeline, id string) (a domain.UserPipelineAccess, err error) {
	a.PipelineID = pipeline
	a.UserProfileID = id

	row := d.Connection.QueryRow("SELECT access, lastupdated FROM userpipelineaccess where pipelineid=? and userprofileid=?", pipeline, id)
	switch err := row.Scan(&a.Access, &a.LastUpdated); err {
	case sql.ErrNoRows:
		log.Info().Msg("userpipelineaccess id was not found")
		return a, err
	case nil:
		log.Info().Msg("userpipelineaccess id was found")
		return a, nil
	default:
		return a, err
	}

	return a, nil
}

func (d SinglestoreChurroDatabase) CreateUserPipelineAccess(a domain.UserPipelineAccess) error {

	INSERT := "INSERT INTO userpipelineaccess(userprofileid, pipelineid, access, lastupdated) values (?, ?, ?, now())"

	stmt, err := d.Connection.Prepare(INSERT)
	if err != nil {
		log.Error().Stack().Err(err).Msg(INSERT)
		return err
	}

	_, err = stmt.Exec(a.UserProfileID, a.PipelineID, a.Access)
	if err != nil {
		log.Error().Stack().Err(err)
	}

	return nil
}

func (d SinglestoreChurroDatabase) UpdateUserPipelineAccess(a domain.UserPipelineAccess) error {

	updateString := "UPDATE userpipelineaccess set access = ?, lastupdated = now() where userprofileid = ? and pipelineid = ?"

	stmt, err := d.Connection.Prepare(updateString)
	if err != nil {
		log.Error().Stack().Err(err).Msg(updateString)
		return err
	}
	_, err = stmt.Exec(a.Access, a.UserProfileID, a.PipelineID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return err
}

func (d SinglestoreChurroDatabase) DeleteAllUserPipelineAccess(pipeline string) error {
	deleteString := "DELETE FROM userpipelineaccess where pipelineid = ?"
	stmt, err := d.Connection.Prepare(deleteString)
	if err != nil {
		log.Error().Stack().Err(err).Msg(deleteString)
		return err
	}
	_, err = stmt.Exec(pipeline)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) DeleteUserPipelineAccess(pipeline, id string) error {
	deleteString := "DELETE FROM userpipelineaccess where pipelineid = ? and userprofileid = ?"

	stmt, err := d.Connection.Prepare(deleteString)
	if err != nil {
		log.Error().Stack().Err(err).Msg(deleteString)
		return err
	}
	_, err = stmt.Exec(pipeline, id)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) CreateUserProfile(u domain.UserProfile) error {
	id := xid.New().String()
	var INSERT = "INSERT INTO userprofile(password, id, lastname, firstname, email, access, lastupdated) values(?,?,?,?,?,?,now())"

	stmt, err := d.Connection.Prepare(INSERT)
	if err != nil {
		log.Error().Stack().Err(err).Msg(INSERT)
		return err
	}
	_, err = stmt.Exec(u.Password, id, u.LastName, u.FirstName, u.Email, u.Access)
	if err != nil {
		log.Error().Stack().Err(err)
	}

	return err
}

func (d SinglestoreChurroDatabase) UpdateUserProfile(u domain.UserProfile) error {

	updateString := "UPDATE userprofile set password = ?, lastname = ?, firstname = ?, email = ?, access = ?, lastupdated = now() where id = ?"
	stmt, err := d.Connection.Prepare(updateString)
	if err != nil {
		log.Error().Stack().Err(err).Msg(updateString)
		return err
	}
	_, err = stmt.Exec(u.Password, u.LastName, u.FirstName, u.Email, u.Access, u.ID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return err
}

func (d SinglestoreChurroDatabase) DeleteUserProfile(id string) (err error) {
	_, err = d.Connection.Exec(fmt.Sprintf("DELETE FROM userprofile where id='%s'", id))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d SinglestoreChurroDatabase) Authenticate(email, password string) (u domain.UserProfile, err error) {
	return u, nil
}
func (d SinglestoreChurroDatabase) GetAllUserProfile() (users []domain.UserProfile, err error) {
	users = make([]domain.UserProfile, 0)

	rows, err := d.Connection.Query("SELECT id, firstname, lastname, password, access, email, lastupdated FROM userprofile")
	if err != nil {
		log.Error().Stack().Err(err)
		return users, err
	}

	for rows.Next() {
		p := domain.UserProfile{}
		err = rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Password, &p.Access, &p.Email, &p.LastUpdated)
		if err != nil {
			log.Error().Stack().Err(err)
			return users, err
		}
		users = append(users, p)
	}

	return users, nil
}

func (d SinglestoreChurroDatabase) GetAllUserProfileForPipeline(pipelineid string) (users []domain.UserProfile, err error) {
	users = make([]domain.UserProfile, 0)

	rows, err := d.Connection.Query("SELECT a.id, a.firstname, a.lastname, a.password, a.access, a.email, a.lastupdated FROM userprofile a, userpipelineaccess b where a.id = b.userprofileid and b.pipelineid = ?", pipelineid)
	if err != nil {
		log.Error().Stack().Err(err)
		return users, err
	}

	for rows.Next() {
		p := domain.UserProfile{}
		err = rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Password, &p.Access, &p.Email, &p.LastUpdated)
		if err != nil {
			log.Error().Stack().Err(err)
			return users, err
		}
		users = append(users, p)
	}

	return users, nil
}
func (d SinglestoreChurroDatabase) GetUserProfileByEmail(email string) (u domain.UserProfile, err error) {
	row := d.Connection.QueryRow("SELECT id, firstname, lastname, password, access, email, lastupdated FROM userprofile where email=?", email)
	switch err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Password, &u.Access, &u.Email, &u.LastUpdated); err {
	case sql.ErrNoRows:
		log.Error().Stack().Msg("userprofile email " + email + " was not found")
		return u, err
	case nil:
		return u, nil
	default:
		return u, err
	}

	return u, nil
}
func (d SinglestoreChurroDatabase) GetUserProfile(id string) (u domain.UserProfile, err error) {
	row := d.Connection.QueryRow("SELECT id, firstname, lastname, password, access, email, lastupdated FROM userprofile where id=?", id)
	switch err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Password, &u.Access, &u.Email, &u.LastUpdated); err {
	case sql.ErrNoRows:
		log.Error().Msg("userprofile id was not found")
		return u, err
	case nil:
		log.Error().Msg("userprofile id was found")
		return u, nil
	default:
		return u, err
	}

	return u, nil
}
func (d SinglestoreChurroDatabase) Bootstrap() (err error) {
	var id string
	bootstrapID := "0000"
	row := d.Connection.QueryRow("SELECT id FROM userprofile where id=?", bootstrapID)
	switch err := row.Scan(&id); err {
	case sql.ErrNoRows:
	case nil:
		return nil
	default:
		return err
	}
	sqlStatement := "INSERT INTO userprofile(id, firstname, lastname, password, access, email, lastupdated) values(?,?,?,?,?,?,now())"

	row = d.Connection.QueryRow(sqlStatement, bootstrapID, "admin", "admin", "admin", "Admin", "admin@admin.org")
	switch err := row.Scan(); err {
	case sql.ErrNoRows:
	case nil:
		return nil
	default:
		return err
	}
	if err != nil {
		return err
	}

	return nil
}
