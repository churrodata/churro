// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package mockdb

import (
	"fmt"

	"github.com/churrodata/churro/internal/domain"
)

const testPipelineName = "testpipeline"
const testPipelineID = "1"

var testUser domain.UserProfile
var testUserPipelineAccess domain.UserPipelineAccess

func init() {
	testUser = domain.UserProfile{
		ID:        "13",
		FirstName: "admin",
		LastName:  "admin",
		Password:  "password",
		Access:    "Read",
		Email:     "admin@admin.org",
	}
	testUserPipelineAccess = domain.UserPipelineAccess{
		UserProfileID: testUser.ID,
		PipelineID:    testPipelineID,
		Access:        "Read",
	}
}

func (d MockChurroDatabase) CreateAuthenticatedUser(u domain.AuthenticatedUser) error {

	return nil
}

func (d MockChurroDatabase) DeleteAuthenticatedUser(id string) (err error) {

	return nil
}

func (d MockChurroDatabase) GetUserPipelineAccess(pipeline, id string) (a domain.UserPipelineAccess, err error) {
	if pipeline != testPipelineName {
		return a, fmt.Errorf("pipeline name not found")
	}
	return testUserPipelineAccess, nil
}

func (d MockChurroDatabase) CreateUserPipelineAccess(a domain.UserPipelineAccess) error {
	return nil
}

func (d MockChurroDatabase) UpdateUserPipelineAccess(a domain.UserPipelineAccess) error {

	return nil
}

func (d MockChurroDatabase) DeleteAllUserPipelineAccess(pipeline string) error {

	return nil
}

func (d MockChurroDatabase) DeleteUserPipelineAccess(pipeline, id string) error {

	return nil
}

func (d MockChurroDatabase) CreateUserProfile(u domain.UserProfile) error {

	return nil
}

func (d MockChurroDatabase) UpdateUserProfile(u domain.UserProfile) error {

	return nil
}

func (d MockChurroDatabase) DeleteUserProfile(id string) (err error) {

	return nil
}

func (d MockChurroDatabase) Authenticate(email, password string) (u domain.UserProfile, err error) {
	return u, nil
}
func (d MockChurroDatabase) GetAllUserProfile() (users []domain.UserProfile, err error) {

	return users, nil
}

func (d MockChurroDatabase) GetAllUserProfileForPipeline(pipelineid string) (users []domain.UserProfile, err error) {
	return users, nil
}

func (d MockChurroDatabase) GetUserProfileByEmail(email string) (u domain.UserProfile, err error) {
	return testUser, nil
}

func (d MockChurroDatabase) GetUserProfile(id string) (u domain.UserProfile, err error) {

	return u, nil
}

func (d MockChurroDatabase) Bootstrap() (err error) {
	return nil

}

func (d MockChurroDatabase) CreateChurroDatabase(dbName string) (err error) {
	return nil

}

func (d MockChurroDatabase) CreateAuthObjects() (err error) {

	return nil
}
