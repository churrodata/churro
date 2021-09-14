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
	"fmt"
	"net"
	"os"
	"time"

	"github.com/churrodata/churro/internal/ctl"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC822
	log.Logger = log.With().Caller().Logger()

	log.Info().Msg("churro-ctl")

	debug := flag.Bool("debug", false, "debug flag")
	log.Info().Msg(fmt.Sprintf("debug set to %v", debug))
	serviceCertPath := flag.String("servicecert", "", "path to service cert files e.g. service.crt")
	dbCertPath := flag.String("dbcert", "", "path to database cert files (e.g. ca.crt)")

	flag.Parse()

	pipeline := os.Getenv("CHURRO_PIPELINE")
	if pipeline == "" {
		log.Error().Stack().Msg("CHURRO_PIPELINE env var is required")
		os.Exit(1)
	}
	ns := os.Getenv("CHURRO_NAMESPACE")
	if ns == "" {
		log.Error().Stack().Msg("CHURRO_NAMESPACE env var is required")
		os.Exit(1)
	}

	_, err := os.Stat(*dbCertPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in dbCertPath")
		os.Exit(1)
	}

	// this gets the Pipeline CR from Kube
	pi, err := pkg.GetPipeline()
	if err != nil {
		panic(err)
	}

	server := ctl.NewCtlServer(ns, true, *serviceCertPath, *dbCertPath, pi)
	creds, err := credentials.NewServerTLSFromFile(server.ServiceCreds.ServiceCrt, server.ServiceCreds.ServiceKey)
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", ctl.DefaultPort)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer(grpc.Creds(creds))

	pb.RegisterCtlServer(s, server)
	if err := s.Serve(lis); err != nil {
		panic(err)
	}

}
