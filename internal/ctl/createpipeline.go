// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package ctl

import "github.com/churrodata/churro/internal/db"

// createPipeline creates the pipeline database
func (s *Server) createPipeline() error {

	pi := s.Pi

	churroDB, err := db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return err
	}

	err = churroDB.GetConnection(s.DBCreds, pi.Spec.AdminDataSource)
	if err != nil {
		return err
	}

	// create the pipeline user
	// create user if not exists pipeline1user;
	err = churroDB.CreateUser(pi.Spec.DataSource.Username, pi.Spec.DataSource.Password)
	if err != nil {
		return err
	}

	// create the pipeline database
	/**
	err = churroDB.CreateObjects(pi.Spec.DataSource.Database)
	if err != nil {
		return err
	}
	*/

	// create pipeline database while connected to the admin user
	err = churroDB.CreatePipelineDatabase(pi.Spec.DataSource.Database)
	if err != nil {
		return err
	}

	// connect as the pipeline user
	err = churroDB.GetConnection(s.DBCreds, pi.Spec.DataSource)
	if err != nil {
		return err
	}

	// create the pipeline objects as the pipeline user
	err = churroDB.CreatePipelineObjects(pi.Spec.DataSource.Database, pi.Spec.DataSource.Username)
	if err != nil {
		return err
	}

	return nil

}
