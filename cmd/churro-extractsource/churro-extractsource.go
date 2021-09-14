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
	"net"
	"os"
	"time"

	"github.com/churrodata/churro/internal/extractsource"
	"github.com/churrodata/churro/pkg"
	"github.com/churrodata/churro/pkg/config"
	pb "github.com/churrodata/churro/rpc/extractsource"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC822

	log.Logger = log.With().Caller().Logger()

	debugFlag := flag.Bool("debug", false, "debug logging")

	serviceCertPath := flag.String("servicecert", "", "path to service creds")
	dbCertPath := flag.String("dbcert", "", "path to database cert files (e.g. ca.crt)")

	flag.Parse()

	svcCreds := config.ServiceCredentials{
		ServiceCrt: *serviceCertPath + "/service.crt",
		ServiceKey: *serviceCertPath + "/service.key",
	}
	err := svcCreds.Validate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in Validate ")
		os.Exit(1)
	}

	ns := os.Getenv("CHURRO_NAMESPACE")
	if ns == "" {
		log.Error().Stack().Msg("CHURRO_NAMESPACE env var is required")
		os.Exit(1)
	}

	_, err = os.Stat(*dbCertPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in dbCertPath")
		os.Exit(1)
	}

	dbCreds := config.DBCredentials{
		CACertPath:      *dbCertPath + "/ca.crt",
		SSLRootKeyPath:  *dbCertPath + "/client.root.key",
		SSLRootCertPath: *dbCertPath + "/client.root.crt",
	}

	userDBCreds := config.DBCredentials{
		CACertPath:  *dbCertPath + "/ca.crt",
		SSLKeyPath:  *dbCertPath + "/client." + ns + ".key",
		SSLCertPath: *dbCertPath + "/client." + ns + ".crt",
	}

	creds, err := credentials.NewServerTLSFromFile(svcCreds.ServiceCrt, svcCreds.ServiceKey)
	if err != nil {
		log.Error().Stack().Err(err).Msg("failed to setup TLS ")
		os.Exit(1)
	}

	pi, err := pkg.GetPipeline()
	if err != nil {
		log.Error().Stack().Err(err).Msg("pipeline not found " + pi.Name)
		os.Exit(1)
	}

	server := extractsource.NewExtractSourceServer(*debugFlag, svcCreds, pi, userDBCreds, dbCreds)

	lis, err := net.Listen("tcp", extractsource.DefaultPort)
	if err != nil {
		log.Error().Stack().Err(err).Msg("failed to listen ")
		os.Exit(1)
	}

	server.StartHarvesting()

	s := grpc.NewServer(grpc.Creds(creds))

	pb.RegisterExtractSourceServer(s, server)
	if err := s.Serve(lis); err != nil {
		log.Error().Stack().Err(err).Msg("failed to serve")
		os.Exit(1)
	}

}
