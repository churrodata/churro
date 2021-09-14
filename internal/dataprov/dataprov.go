// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package dataprov holds the data provenance logic which is essentially
// used to identify each data source uniquely with a generated ID
// that is passed with all processed data so users can track data back
// to its source
package dataprov

import (
	"time"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg/config"
	"github.com/rs/xid"
)

// Register a new data provenance instance, return an error
// if it can not be registered with churro
func Register(dp *domain.DataProvenance, pipeline v1alpha1.Pipeline, dbCreds config.DBCredentials) (err error) {

	dp.LastUpdated = time.Now()
	dp.ID = xid.New().String()
	// register the id with the churro data store

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(pipeline.Spec.DatabaseType)
	if err != nil {
		return err
	}

	err = churroDB.GetConnection(dbCreds, pipeline.Spec.DataSource)
	if err != nil {
		return err
	}

	err = churroDB.CreateDataprov(*dp)

	return err
}
