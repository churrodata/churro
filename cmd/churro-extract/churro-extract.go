// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package main

import (
	"flag"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/churrodata/churro/internal/extract"
	"github.com/churrodata/churro/pkg"
	cfg "github.com/churrodata/churro/pkg/config"
)

//var queueMax = 100

func main() {
	zerolog.TimeFieldFormat = time.RFC822

	log.Logger = log.With().Caller().Logger()

	log.Info().Msg("churro-extract")
	serviceCertPath := flag.String("servicecert", "", "path to service cert file e.g. service.crt")

	dbCertPath := flag.String("dbcert", "", "path to database cert files (e.g. ca.crt)")

	debug := flag.Bool("debug", false, "debug flag")

	flag.Parse()

	_, err := os.Stat(*dbCertPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error dbCertPath")
		os.Exit(1)
	}
	pipeline := os.Getenv("CHURRO_NAMESPACE")
	if pipeline == "" {
		log.Error().Stack().Msg("error CHURRO_NAMESPACE env var not set")
		os.Exit(1)
	}
	fileName := os.Getenv("CHURRO_FILENAME")
	if fileName == "" {
		log.Error().Stack().Msg("error CHURRO_FILENAME not set")
		os.Exit(1)
	}
	schemeValue := os.Getenv("CHURRO_SCHEME")
	if schemeValue == "" {
		log.Error().Stack().Msg("CHURRO_SCHEME env var is required")
		os.Exit(1)
	}
	tableName := os.Getenv("CHURRO_TABLENAME")
	if tableName == "" {
		log.Error().Stack().Msg("CHURRO_TABLENAME env var is required")
		os.Exit(1)
	}
	watchDirName := os.Getenv("CHURRO_WATCHDIR_NAME")
	if watchDirName == "" {
		log.Error().Stack().Msg("CHURRO_WATCHDIR_NAME env var is required")
		os.Exit(1)
	}

	log.Info().Msg("CHURRO_TABLENAME " + tableName)
	log.Info().Msg("CHURRO_SCHEME " + schemeValue)
	log.Info().Msg("CHURRO_FILENAME " + fileName)
	log.Info().Msg("CHURRO_NAMESPACE " + pipeline)
	log.Info().Msg("CHURRO_WATCHDIR_NAME " + watchDirName)

	dbCreds := cfg.DBCredentials{
		CAKeyPath:       *dbCertPath + "/ca.key",
		CACertPath:      *dbCertPath + "/ca.crt",
		SSLRootKeyPath:  *dbCertPath + "/client." + "root" + ".key",
		SSLRootCertPath: *dbCertPath + "/client." + "root" + ".crt",
		SSLKeyPath:      *dbCertPath + "/client." + pipeline + ".key",
		SSLCertPath:     *dbCertPath + "/client." + pipeline + ".crt",
	}
	svcCreds := cfg.ServiceCredentials{
		ServiceCrt: *serviceCertPath + "/service.crt",
		ServiceKey: *serviceCertPath + "/service.key",
	}

	err = svcCreds.Validate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error svccreds validate ")
		os.Exit(1)
	}
	err = dbCreds.Validate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error dbcreds validate ")
		os.Exit(1)
	}

	pi, err := pkg.GetPipeline()
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not get pipeline " + pi.Name)
		os.Exit(1)
	}

	extract.NewExtractServer(fileName, schemeValue, tableName, *debug, svcCreds, dbCreds, pi)
	log.Info().Msg("extract ending...")

}
