// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package extract

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/internal/dataprov"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
)

const DEFAULT_HTTPPOST_PORT = "10000"

type httppostwrapper struct {
	URLEncoded bool
}

// ExtractHTTPPost listen for any http posts
func (s *Server) ExtractHTTPPost(ctx context.Context) (err error) {

	log.Info().Msg("ExtractHTTPPost ...api URL " + s.ExtractSource.Path)

	// register to dataprov
	dp := domain.DataProvenance{
		Name: s.FileName,
		Path: s.FileName,
	}
	err = dataprov.Register(&dp, s.Pi, s.DBCreds)
	if err != nil {
		log.Error().Stack().Err(err).Msg("can not register data prov")
		os.Exit(1)
	}
	log.Info().Msg(fmt.Sprintf("dp info %+v", dp))

	// initialize with the single message default
	jsonStruct := extractapi.RawFormat{
		ColumnNames: []string{"metadata"},
		ColumnTypes: []string{"jsonb"},
	}

	// update if we have extract rules defined
	if len(s.ExtractSource.ExtractRules) > 0 {
		//jsonStruct.Columns = getColumns(s.ExtractSource)
		//jsonStruct.ColumnNames = getColumnNames(jsonStruct.Columns)
		//jsonStruct.ColumnTypes = getColumnTypes(jsonStruct.Columns)
	}

	log.Info().Msg(fmt.Sprintf("jsonStruct at top is %+v", jsonStruct))
	jsonStruct.Path = dp.Path
	jsonStruct.Dataprov = dp.ID

	//apiurl := s.ExtractSource.Path

	jobProfile := domain.JobProfile{
		ID:               os.Getenv("CHURRO_EXTRACTLOG"),
		JobName:          os.Getenv("POD_NAME"),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		DataProvenanceID: dp.ID,
		FileName:         s.FileName,
		TableName:        s.ExtractSource.Tablename,
		RecordsLoaded:    0,
	}

	log.Info().Msg("setting table name to " + s.ExtractSource.Tablename)
	err = s.insertJobProfile(jobProfile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error inserting initial JobProfile")
		return err
	}
	log.Info().Msg("inserted Extractlog")

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating the database")
		return err
	}
	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		return err
	}

	err = s.tableCheck(jsonStruct.ColumnNames, jsonStruct.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error during tableCheck ")
		return err
	}

	if len(s.ExtractSource.ExtractRules) > 0 {

		jsonStruct.Columns = make([]extractapi.Column, 0)
		jsonStruct.ColumnNames = make([]string, 0)
		jsonStruct.ColumnTypes = make([]string, 0)

		//allCols := make([][]interface{}, 0)
		//var rows int
		log.Info().Msg(fmt.Sprintf("user has %d extract rules defined\n", len(s.ExtractSource.ExtractRules)))
		for _, r := range s.ExtractSource.ExtractRules {

			ec := extractapi.Column{
				Name: r.ColumnName,
				Path: r.ColumnPath,
				Type: r.ColumnType,
			}
			jsonStruct.Columns = append(jsonStruct.Columns, ec)
			jsonStruct.ColumnNames = append(jsonStruct.ColumnNames, r.ColumnName)
			jsonStruct.ColumnTypes = append(jsonStruct.ColumnTypes, r.ColumnType)
		}
	}

	port := DEFAULT_HTTPPOST_PORT

	var portValue int64
	portValue, err = strconv.ParseInt(port, 10, 32)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error parsing port string")
		return err
	}

	err = createService(s.Pi.Name, s.ExtractSource.Name, int32(portValue), s.ExtractSource.Servicetype)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating Service")
		return err
	}

	// listen for posts

	u := httppostwrapper{
		URLEncoded: true,
	}

	r := mux.NewRouter()

	r.HandleFunc("/extractsourcepost", u.ExtractSourceHTTPPost).Methods("POST")

	secure := false

	if secure {
		log.Info().Msg("transport https")
		err = http.ListenAndServeTLS(":"+port, "./certs/ui/https-server.crt", "./certs/ui/https-server.key", r)

	} else {
		log.Info().Msg("transport http")
		err = http.ListenAndServe(":"+port, r)

	}
	if err != nil {
		log.Fatal().Err(err).Msg("error in listen")
		return err
	}

	return nil
}

func (u *httppostwrapper) ExtractSourceHTTPPost(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("ExtractSourceHTTPPost called")
	fmt.Printf("http request %+v\n", r)
	if u.URLEncoded {
		r.ParseForm()

		firstname := r.Form["firstname"][0]
		log.Info().Msg(firstname)
	}

}

func createService(pipelineName, extractSourceName string, port int32, serviceType string) (err error) {

	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error getting Kube clientset")
		return err
	}

	// see if the service is already created
	serviceExists := true
	getOptions := metav1.GetOptions{}
	_, err = clientset.CoreV1().Services(pipelineName).Get(context.TODO(), extractSourceName, getOptions)
	if kerrors.IsNotFound(err) {
		serviceExists = false
		log.Info().Msg("httppost service " + extractSourceName + " does not exist")
	} else if err != nil {
		log.Error().Stack().Err(err).Msg("error getting Service")
		return err
	}

	// create the service if necessary
	if !serviceExists {
		service := v1.Service{}
		service.ObjectMeta.Labels = map[string]string{"app": "churro", "service": "churro-extract", "extractsourcename": extractSourceName}
		service.Name = extractSourceName
		service.Namespace = pipelineName
		sp := v1.ServicePort{
			Name: "http",
			Port: port,
		}
		service.Spec.Ports = []v1.ServicePort{sp}
		service.Spec.Selector = map[string]string{"app": "churro", "service": "churro-extract", "extractsourcename": extractSourceName}
		service.Spec.Type = v1.ServiceTypeClusterIP
		if serviceType == "LoadBalancer" {
			service.Spec.Type = v1.ServiceTypeLoadBalancer
		}
		/**
		  apiVersion: v1
		  kind: Service
		  metadata:
		    name: serviceName
		    labels:
		      app: churro
		      service: churro-extract
		      extractsourcename: extractSourceName
		  spec:
		    ports:
		    - port: 10000
		      targetPort: 10000
		      name: http
		    selector:
		      app: churro
		      service: churro-extract
		      extractsourcename: extractSourceName
		*/
		_, err := clientset.CoreV1().Services(pipelineName).Create(context.TODO(), &service, metav1.CreateOptions{})
		if err != nil {
			log.Error().Stack().Err(err).Msg("error creating httppost Service " + extractSourceName)
			return err
		}
		log.Info().Msg("created Service " + extractSourceName)

		return nil

	}
	return nil
}
