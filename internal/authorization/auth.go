// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package authorization

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/churrodata/churro/internal/db"
)

// Bitmask ...
type Bitmask uint32

// HasFlag ...
func (f Bitmask) HasFlag(flag Bitmask) bool { return f&flag != 0 }

// AddFlag ...
func (f *Bitmask) AddFlag(flag Bitmask) { *f |= flag }

// ClearFlag ...
func (f *Bitmask) ClearFlag(flag Bitmask) { *f &= ^flag }

// ToggleFlag ...
func (f *Bitmask) ToggleFlag(flag Bitmask) { *f ^= flag }

const (
	// ActionRead ...
	ActionRead Bitmask = 1 << iota
	// ActionWrite ...
	ActionWrite
	// ActionAdmin ...
	ActionAdmin
)

// ObjectPipeline ...
const ObjectPipeline = "pipeline"

// AuthMap ... jemccormi, pipeline, TESTFLAG_ONE|TESTFLAG_THREE
type AuthMap struct {
	ID          string    `json:"id"`
	LastUpdated time.Time `json:"lastupdated"`
	Subject     string    `json:"subject"`
	PipelineID  string    `json:"pipelineid"`
	Object      string    `json:"object"`
	Action      Bitmask   `json:"action"`
}

// Authorized ....
func (m AuthMap) Authorized(dbType string) bool {
	am, err := getAuthMap(m.Subject, m.PipelineID, dbType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in getAuthMap ")
		return false
	}

	// right now, Object is only a 'pipeline'

	am.Object = m.Object

	log.Info().Msg(fmt.Sprintf("Authorized values %+v", am))

	//allow an ADMIN to do anything
	if m.Subject == am.Subject &&
		am.Action.HasFlag(ActionAdmin) {
		return true
	}

	log.Info().Msg("here we pass by the admin check!")

	if m.Subject == am.Subject &&
		m.Object == am.Object {
		if am.Action.HasFlag(m.Action) {
			return true
		}
		// allow a user with WRITE to do a READ
		if am.Action.HasFlag(ActionWrite) &&
			m.Action == ActionRead {
			return true
		}
	}
	return false
}

func getAuthMap(subject, pipelineID, dbType string) (AuthMap, error) {

	am := AuthMap{}

	churroDB, err := db.NewChurroDB(dbType)
	if err != nil {
		return am, err
	}

	err = churroDB.GetConnection(AdminDB.DBCreds, AdminDB.Source)
	if err != nil {
		return am, err
	}

	up, err := churroDB.GetUserProfileByEmail(subject)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return am, err
	}

	am.Subject = up.Email

	// admin has full access
	if up.Access == "Admin" {
		am.Action = ActionAdmin
		return am, err
	}

	if pipelineID != "" {
		upa, err := churroDB.GetUserPipelineAccess(pipelineID, up.ID)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return am, err
		}
		up.Access = upa.Access
	}

	switch up.Access {
	case "Read":
		am.Action = ActionRead
	case "Write":
		am.Action = ActionWrite
	case "Admin":
		am.Action = ActionAdmin
	}
	return am, nil

}
