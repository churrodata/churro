// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package cockroachdb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/xid"
)

func (d CockroachChurroDatabase) CreateAuthenticatedUser(u domain.AuthenticatedUser) error {
	u.ID = xid.New().String()
	INSERT := "INSERT INTO authenticateduser(id, token, locked, lastupdated) values($1,$2,$3,now()) returning id"

	err := d.Connection.QueryRow(INSERT, u.ID, u.Token, u.Locked).Scan(&u.ID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) DeleteAuthenticatedUser(id string) (err error) {
	_, err = d.Connection.Exec(fmt.Sprintf("DELETE FROM authenticateduser where id='%s'", id))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) GetUserPipelineAccess(pipeline, id string) (a domain.UserPipelineAccess, err error) {
	a.PipelineID = pipeline
	a.UserProfileID = id

	row := d.Connection.QueryRow("SELECT access, lastupdated FROM userpipelineaccess where pipelineid=$1 and userprofileid=$2", pipeline, id)
	switch err := row.Scan(&a.Access, &a.LastUpdated); err {
	case sql.ErrNoRows:
		log.Error().Stack().Err(err).Msg("userpipelineaccess id was not found")
		return a, err
	case nil:
		log.Info().Msg("userpipelineaccess id was found")
		return a, nil
	default:
		return a, err
	}
	return a, nil
}

func (d CockroachChurroDatabase) CreateUserPipelineAccess(a domain.UserPipelineAccess) error {
	var INSERT = "INSERT INTO userpipelineaccess(userprofileid, pipelineid, access, lastupdated) values($1,$2,$3,now()) returning userprofileid"

	err := d.Connection.QueryRow(INSERT, a.UserProfileID, a.PipelineID, a.Access).Scan(&a.UserProfileID)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) UpdateUserPipelineAccess(a domain.UserPipelineAccess) error {
	datetime := time.Now()

	_, err := d.Connection.Exec(
		fmt.Sprintf("UPDATE userpipelineaccess set (access, lastupdated) = ('%s','%v') where userprofileid = '%s' and pipelineid = '%s'",
			a.Access,
			datetime.Format("2006-01-02T15:04:05.999999999"),
			a.UserProfileID,
			a.PipelineID))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) DeleteAllUserPipelineAccess(pipeline string) error {
	_, err := d.Connection.Exec(fmt.Sprintf("DELETE FROM UserPipelineAccess where pipelineid = '%s'", pipeline))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) DeleteUserPipelineAccess(pipeline, id string) error {
	_, err := d.Connection.Exec(fmt.Sprintf("DELETE FROM UserPipelineAccess where pipelineid = '%s' and userprofileid = '%s'", pipeline, id))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) CreateUserProfile(u domain.UserProfile) error {
	id := xid.New().String()
	var INSERT = "INSERT INTO userprofile(password, id, lastname, firstname, email, access, lastupdated) values($1,$2,$3,$4,$5,$6,now()) returning id"

	err := d.Connection.QueryRow(INSERT, u.Password, id, u.LastName, u.FirstName, u.Email, u.Access).Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) UpdateUserProfile(u domain.UserProfile) error {
	datetime := time.Now()

	sqlStr := fmt.Sprintf("UPDATE userprofile set (password, lastname, firstname, email, access, lastupdated) = ('%s','%s','%s','%s','%s','%v') where id = '%s'", u.Password, u.LastName, u.FirstName, u.Email, u.Access, datetime.Format("2006-01-02T15:04:05.999999999"), u.ID)
	log.Info().Msg("sqlStr is " + sqlStr)
	_, err := d.Connection.Exec(sqlStr)
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) DeleteUserProfile(id string) (err error) {
	_, err = d.Connection.Exec(fmt.Sprintf("DELETE FROM userprofile where id='%s'", id))
	if err != nil {
		log.Error().Stack().Err(err)
		return err
	}

	return nil
}

func (d CockroachChurroDatabase) Authenticate(email, password string) (u domain.UserProfile, err error) {
	return u, nil
}
func (d CockroachChurroDatabase) GetAllUserProfile() (users []domain.UserProfile, err error) {
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

func (d CockroachChurroDatabase) GetAllUserProfileForPipeline(pipelineid string) (users []domain.UserProfile, err error) {
	users = make([]domain.UserProfile, 0)

	rows, err := d.Connection.Query("SELECT a.id, a.firstname, a.lastname, a.password, a.access, a.email, a.lastupdated FROM userprofile a, userpipelineaccess b where a.id = b.userprofileid and b.pipelineid = $1", pipelineid)
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

func (d CockroachChurroDatabase) GetUserProfileByEmail(email string) (u domain.UserProfile, err error) {
	row := d.Connection.QueryRow("SELECT id, firstname, lastname, password, access, email, lastupdated FROM userprofile where email=$1", email)
	switch err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Password, &u.Access, &u.Email, &u.LastUpdated); err {
	case sql.ErrNoRows:
		log.Error().Stack().Err(err).Msg("userprofile email was not found" + email)
		return u, err
	case nil:
		return u, nil
	default:
		return u, err
	}

	return u, nil
}
func (d CockroachChurroDatabase) GetUserProfile(id string) (u domain.UserProfile, err error) {
	row := d.Connection.QueryRow("SELECT id, firstname, lastname, password, access, email, lastupdated FROM userprofile where id=$1", id)
	switch err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Password, &u.Access, &u.Email, &u.LastUpdated); err {
	case sql.ErrNoRows:
		log.Error().Stack().Err(err).Msg("userprofile id was not found")
		return u, err
	case nil:
		log.Info().Msg("userprofile id was found")
		return u, nil
	default:
		return u, err
	}

	return u, nil
}

func (d CockroachChurroDatabase) Bootstrap() (err error) {
	var id string
	bootstrapID := "0000"
	row := d.Connection.QueryRow("SELECT id FROM userprofile where id=$1", bootstrapID)
	switch err := row.Scan(&id); err {
	case sql.ErrNoRows:
	case nil:
		return nil
	default:
		return err
	}
	sqlStatement := "INSERT INTO userprofile(id, firstname, lastname, password, access, email, lastupdated) values($1,$2,$3,$4,$5,$6,now()) returning id"

	err = d.Connection.QueryRow(sqlStatement, bootstrapID, "admin", "admin", "admin", "Admin", "admin@admin.org").Scan(&id)
	if err != nil {
		return err
	}
	return nil

}

func (d CockroachChurroDatabase) CreateChurroDatabase(dbName string) (err error) {
	// make sure churro admin database is created
	sqlStr := fmt.Sprintf("CREATE DATABASE if not exists %s", dbName)
	_, err = d.Connection.Exec(sqlStr)
	log.Info().Msg(sqlStr)
	if err != nil {
		return err
	}
	log.Info().Msg("Successfully created database" + dbName)
	return nil

}

func (d CockroachChurroDatabase) CreateAuthObjects() (err error) {

	// create AuthenticatedUser
	_, err = d.Connection.Exec("CREATE TABLE if not exists authenticateduser (id VARCHAR(255) PRIMARY KEY, token VARCHAR(64) NOT NULL, locked boolean NOT NULL, lastupdated TIMESTAMP NULL)")
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
