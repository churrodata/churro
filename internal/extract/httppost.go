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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Encoding       string
	RawJSONMessage bool
	CSVStruct      extractapi.CSVFormat
	Server         *Server
}

// ExtractHTTPPost listen for any http posts
func (s *Server) ExtractHTTPPost(ctx context.Context) (err error) {

	log.Info().Msg("ExtractHTTPPost ...api URL " + s.ExtractSource.Path)

	u := httppostwrapper{
		Encoding: s.ExtractSource.Encoding,
		Server:   s,
	}
	log.Info().Msg("setting encoding to " + u.Encoding)

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

	csvStruct := extractapi.CSVFormat{}

	log.Info().Msg(fmt.Sprintf("csvStruct at top is %+v", csvStruct))
	csvStruct.Path = dp.Path
	csvStruct.Dataprov = dp.ID

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

	csvStruct.Tablename = s.TableName

	// pull out all the extract rules column information
	if len(s.ExtractSource.ExtractRules) > 0 {

		csvStruct.Columns = make([]extractapi.Column, 0)
		csvStruct.ColumnNames = make([]string, 0)
		csvStruct.ColumnTypes = make([]string, 0)

		//allCols := make([][]interface{}, 0)
		//var rows int
		log.Info().Msg(fmt.Sprintf("user has %d extract rules defined\n", len(s.ExtractSource.ExtractRules)))
		for _, r := range s.ExtractSource.ExtractRules {

			ec := extractapi.Column{
				Name: r.ColumnName,
				Path: r.ColumnPath,
				Type: r.ColumnType,
			}
			csvStruct.Columns = append(csvStruct.Columns, ec)
			csvStruct.ColumnNames = append(csvStruct.ColumnNames, r.ColumnName)
			csvStruct.ColumnTypes = append(csvStruct.ColumnTypes, r.ColumnType)
		}
	} else if len(s.ExtractSource.ExtractRules) == 0 && s.ExtractSource.Encoding == "json" {
		//this means we will store the httppost as a raw json message
		u.RawJSONMessage = true
		csvStruct.ColumnNames = []string{"metadata"}
		csvStruct.ColumnTypes = []string{"jsonb"}
	}

	u.CSVStruct = csvStruct
	u.CSVStruct.PipelineName = s.Pi.Name

	// see if the database table has been created, create it if not
	err = s.tableCheck(csvStruct.ColumnNames, csvStruct.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error during tableCheck ")
		return err
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
	log.Info().Msg(fmt.Sprintf("columns %+v", u.CSVStruct.Columns))
	fmt.Printf("http request %+v\n", r)
	log.Info().Msg("u.Encoding here is " + u.Encoding)
	if u.Encoding == "urlencoded" {
		r.ParseForm()

		u.CSVStruct.Records = getRowFromForm(r.Form, u.CSVStruct)
		log.Info().Msg(fmt.Sprintf("records is %+v", u.CSVStruct.Records))
	} else {
		// json case
		if u.RawJSONMessage {
			//assume a single column of jsonb type is the desired action
		} else {
			// assume jsonpath columns to be extracted
		}
	}

	/**
	err := transform.RunRules(u.CSVStruct.ColumnNames, u.CSVStruct.Records[0].Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in RunRules ")
	}
	log.Info().Msg(fmt.Sprintf("after transform %+v", xmlStruct.Records[i].Cols))
	*/

	httppostBytes, _ := json.Marshal(u.CSVStruct)
	msg := extractapi.LoaderMessage{}
	msg.Metadata = httppostBytes
	msg.DataFormat = "httppost"
	jobProfile := domain.JobProfile{
		ID:               os.Getenv("CHURRO_EXTRACT"),
		JobName:          os.Getenv("POD_NAME"),
		StartDate:        time.Now().Format("2006-01-02 15:04:05"),
		DataProvenanceID: u.CSVStruct.Dataprov,
		FileName:         "",
		RecordsLoaded:    1,
	}

	churroDB, err := db.NewChurroDB(u.Server.Pi.Spec.DatabaseType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating the database")
		os.Exit(1)
	}

	err = churroDB.GetConnection(u.Server.DBCreds, u.Server.Pi.Spec.AdminDataSource)

	if err != nil {
		log.Error().Stack().Err(err).Msg("error connecting to the database")
		os.Exit(1)
	}

	err = churroDB.CreateExtractLog(jobProfile)
	if err != nil {
		log.Error().Stack().Err(err)
	}

	u.Server.process(jobProfile, churroDB, u.Server.Pi.Spec.DataSource.Database, msg)
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

// getRowFromForm parses out a single row from the http form
func getRowFromForm(form url.Values, raw extractapi.CSVFormat) (row []extractapi.GenericRow) {
	//cols := make([][]string, 1)

	thisrow := extractapi.GenericRow{}
	thisrow.Key = time.Now().UnixNano()

	for i := 0; i < len(raw.Columns); i++ {
		c := raw.Columns[i]
		//firstname := r.Form["firstname"][0]
		log.Info().Msg("would extract from url form name:" + c.Name + " path:" + c.Path + " type:" + c.Type)
		val := form[c.Path][0]
		log.Info().Msg("extracted " + val + " of type " + c.Type)
		thisrow.Cols = append(thisrow.Cols, val)
	}

	row = append(row, thisrow)

	return row
}
