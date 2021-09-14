// Copyright 2021 churrodata LLC
// Author: djm

package authorization

import (
	"database/sql"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/pkg/config"
)

// AdminDatabase ...
type AdminDatabase struct {
	DBPath           string
	ConnectionString string
	db               *sql.DB
	Created          bool
	DBCreds          config.DBCredentials
	Source           v1alpha1.Source
}

// AdminDB this is the single instance of the db
var AdminDB AdminDatabase
