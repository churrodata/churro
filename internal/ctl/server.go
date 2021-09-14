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

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg/config"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DefaultPort is the ctl server port
const DefaultPort = ":8088"

// Server defines the ctl server configuation
type Server struct {
	Pi           v1alpha1.Pipeline
	ServiceCreds config.ServiceCredentials
	DBCreds      config.DBCredentials
	UserDBCreds  config.DBCredentials
}

func init() {
	log.Info().Msg("initializing Pipelines")
}

// NewCtlServer constructs a ctl server based on the passed
// configuration, a pointer to the server is returned
func NewCtlServer(ns string, debug bool, serviceCertPath string, dbCertPath string, pipeline v1alpha1.Pipeline) *Server {
	s := Server{
		Pi: pipeline,
	}

	err := s.SetupCredentials(ns, serviceCertPath, dbCertPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting credentials")
		os.Exit(1)
	}

	// loop a few times to give the database a chance to start up
	dbErrorsMax := 7
	for i := 0; i < dbErrorsMax; i++ {
		err = s.verify()
		if err != nil {
			if dbErrorsMax == 0 {
				log.Error().Stack().Err(err).Msg("error seeding pipeline admin database")
				os.Exit(1)
			}
			dbErrorsMax--
			log.Error().Stack().Err(err).Msg("error in verifying database, will retry in 10 seconds")
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}

	err = s.createPipeline()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error seeding pipeline database")
		os.Exit(1)
	}

	return &s
}

// Ping ....
func (s *Server) Ping(ctx context.Context, request *pb.PingRequest) (response *pb.PingResponse, err error) {
	if false {
		return nil, status.Errorf(codes.InvalidArgument,
			"something is not right")
	}

	return &pb.PingResponse{}, nil
}

// UnimplementedCtlServer ...
func UnimplementedCtlServer() {
}

// SetupCredentials ...
func (s *Server) SetupCredentials(ns, serviceCertPath, dbCertPath string) (err error) {
	s.ServiceCreds = config.ServiceCredentials{
		ServiceCrt: serviceCertPath + "/service.crt",
		ServiceKey: serviceCertPath + "/service.key",
	}

	// churro-ctl uses the database root credentials
	s.DBCreds = config.DBCredentials{
		CACertPath:      dbCertPath + "/ca.crt",
		CAKeyPath:       dbCertPath + "/ca.key",
		SSLRootKeyPath:  dbCertPath + "/client.root.key",
		SSLRootCertPath: dbCertPath + "/client.root.crt",
		SSLKeyPath:      dbCertPath + "/client." + ns + ".key",
		SSLCertPath:     dbCertPath + "/client." + ns + ".crt",
	}

	s.UserDBCreds = config.DBCredentials{
		CACertPath:      dbCertPath + "/ca.crt",
		CAKeyPath:       dbCertPath + "/ca.key",
		SSLRootKeyPath:  dbCertPath + "/client.root.key",
		SSLRootCertPath: dbCertPath + "/client.root.crt",
		SSLKeyPath:      dbCertPath + "/client." + ns + ".key",
		SSLCertPath:     dbCertPath + "/client." + ns + ".crt",
	}

	err = s.ServiceCreds.Validate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in svccreds validate")
		return err
	}

	err = s.DBCreds.Validate()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in dbcreds validte")
		return err
	}

	return nil
}

func (s *Server) verify() error {

	cfg := s.Pi.Spec.AdminDataSource
	churroDB, err := db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return err
	}

	log.Info().Msg(fmt.Sprintf("connecting with admin source %+v\n", s.Pi.Spec.AdminDataSource))
	//TODO refactor this hack into the db layer
	if s.Pi.Spec.DatabaseType == domain.DatabaseMysql {
		cfg.Database = domain.DatabaseMysql
	}
	if s.Pi.Spec.DatabaseType == domain.DatabaseSinglestore {
		cfg.Database = "memsql"
	}
	err = churroDB.GetConnection(s.DBCreds, cfg)
	if err != nil {
		return err
	}

	log.Debug().Msg("logged on as db churro admin user " + cfg.Username)

	// make sure churro admin database is created

	//TODO refactor this hack into the db layer, this resets
	// the database to churro instead of the mysql database and
	// is only done to bootstrap the churro database
	if s.Pi.Spec.DatabaseType == domain.DatabaseMysql {
		cfg.Database = s.Pi.Spec.AdminDataSource.Database
	}
	if s.Pi.Spec.DatabaseType == domain.DatabaseSinglestore {
		cfg.Database = "churro"
	}
	err = churroDB.CreateObjects(cfg.Database)
	if err != nil {
		return err
	}

	return nil
}
