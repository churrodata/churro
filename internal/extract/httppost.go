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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ohler55/ojg/oj"
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
	CSVStruct      extractapi.GenericFormat
	Server         *Server
	DP             domain.DataProvenance
}

// ExtractHTTPPost listen for any http posts
func (s *Server) ExtractHTTPPost(ctx context.Context) (err error) {

	log.Info().Msg("ExtractHTTPPost ...api URL " + s.ExtractSource.Path)

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

	u := httppostwrapper{
		Encoding: s.ExtractSource.Encoding,
		Server:   s,
		DP:       dp,
	}
	log.Info().Msg("setting encoding to " + u.Encoding)

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

	/**
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
	*/

	u.CSVStruct = extractapi.GenericFormat{
		Path:      u.DP.Path,
		Dataprov:  u.DP.ID,
		Tablename: s.ExtractSource.Tablename,
	}

	u.CSVStruct.Columns = make([]extractapi.Column, 0)
	u.CSVStruct.ColumnNames = make([]string, 0)
	u.CSVStruct.ColumnTypes = make([]string, 0)

	// pull out all the extract rules column information
	if len(s.ExtractSource.ExtractRules) > 0 {

		//allCols := make([][]interface{}, 0)
		//var rows int
		log.Info().Msg(fmt.Sprintf("user has %d extract rules defined\n", len(s.ExtractSource.ExtractRules)))
		for _, r := range s.ExtractSource.ExtractRules {

			ec := extractapi.Column{
				Name: r.ColumnName,
				Path: r.ColumnPath,
				Type: r.ColumnType,
			}
			u.CSVStruct.Columns = append(u.CSVStruct.Columns, ec)
			u.CSVStruct.ColumnNames = append(u.CSVStruct.ColumnNames, r.ColumnName)
			u.CSVStruct.ColumnTypes = append(u.CSVStruct.ColumnTypes, r.ColumnType)
		}
	} else if len(s.ExtractSource.ExtractRules) == 0 && s.ExtractSource.Encoding == "json" {
		//this means we will store the httppost as a raw json message
		u.RawJSONMessage = true
		u.CSVStruct.ColumnNames = []string{"metadata"}
		u.CSVStruct.ColumnTypes = []string{"jsonb"}
	}

	u.CSVStruct.PipelineName = s.Pi.Name

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

	// see if the database table has been created, create it if not
	err = u.Server.tableCheck(u.CSVStruct.ColumnNames, u.CSVStruct.ColumnTypes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error during tableCheck ")
		return
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
	msg := extractapi.LoaderMessage{}
	var err error
	var someBytes []byte

	if u.Encoding == "urlencoded" {
		r.ParseForm()

		someBytes, err = u.getRowFromForm(r.Form)
		if err != nil {
			return
		}
		log.Info().Msg(fmt.Sprintf("records is %+v", u.CSVStruct.Records))
		//TODO fix this duplication below
		msg.DataFormat = extractapi.HTTPPostScheme
		u.Server.SchemeValue = extractapi.HTTPPostScheme
	} else {

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error reading json from body")
			return
		}
		log.Info().Msg(fmt.Sprintf("json request body is %s", string(b)))
		// json case
		if u.RawJSONMessage {
			msg.DataFormat = extractapi.JSONScheme
			u.Server.SchemeValue = extractapi.JSONScheme
			//assume a single column of jsonb type is the desired action
			someBytes, err = u.getRawRowFromJSON(b)
			if err != nil {
				return
			}
			log.Info().Msg("raw json message to be processed")
		} else {
			msg.DataFormat = extractapi.JSONPathScheme
			u.Server.SchemeValue = extractapi.JSONPathScheme
			// assume jsonpath columns to be extracted
			someBytes, err = u.getRowFromJSON(string(b))
			if err != nil {
				return
			}
			log.Info().Msg(fmt.Sprintf("records is %+v", u.CSVStruct.Records))
			log.Info().Msg("extracting from json message")
		}
	}

	/**
	err := transform.RunRules(u.CSVStruct.ColumnNames, u.CSVStruct.Records[0].Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in RunRules ")
	}
	log.Info().Msg(fmt.Sprintf("after transform %+v", xmlStruct.Records[i].Cols))
	*/

	//httppostBytes, _ := json.Marshal(u.CSVStruct)
	msg.Metadata = someBytes
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

// getRowFromJSON parses out a single row from a single JSON message
func (u *httppostwrapper) getRowFromJSON(jsonMessage string) (someBytes []byte, err error) {

	row := make([]extractapi.GenericRow, 0)
	obj, parseError := oj.ParseString(jsonMessage)
	if parseError != nil {
		log.Error().Err(parseError).Stack().Msg("jsonpath parse error")
		return someBytes, parseError
	}

	allCols := make([][]interface{}, 0)

	for _, r := range u.Server.ExtractSource.ExtractRules {

		cols, err := getJSONPathColumns(obj, r.ColumnPath)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return someBytes, err
		}
		log.Info().Msg(fmt.Sprintf("jeff cols here is %+v", cols))
		allCols = append(allCols, cols)
	}

	//for row := 0; row < rows; row++ {
	thisrow := extractapi.GenericRow{
		Key: time.Now().UnixNano(),
	}
	thisrow.Cols = make([]interface{}, 0)
	for cell := 0; cell < len(allCols); cell++ {
		thisrow.Cols = append(thisrow.Cols, allCols[cell][0])
	}

	/**
	err := transform.RunRules(jsonStruct.ColumnNames, r.Cols, s.ExtractSource.ExtractRules, s.TransformFunctions)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in run rules")
	}
	*/
	//u.CSVStruct.Records = append(u.CSVStruct.Records, r)
	//}
	row = append(row, thisrow)
	u.CSVStruct.Records = row

	someBytes, err = json.Marshal(u.CSVStruct)

	return someBytes, nil
}

// getRowFromForm parses out a single row from the http form
func (u *httppostwrapper) getRowFromForm(form url.Values) (someBytes []byte, err error) {

	//cols := make([][]string, 1)

	row := make([]extractapi.GenericRow, 0)
	thisrow := extractapi.GenericRow{}
	thisrow.Key = time.Now().UnixNano()

	for i := 0; i < len(u.CSVStruct.Columns); i++ {
		c := u.CSVStruct.Columns[i]
		//firstname := r.Form["firstname"][0]
		log.Info().Msg("would extract from url form name:" + c.Name + " path:" + c.Path + " type:" + c.Type)
		val := form[c.Path][0]
		log.Info().Msg("extracted " + val + " of type " + c.Type)
		thisrow.Cols = append(thisrow.Cols, val)
	}

	row = append(row, thisrow)

	u.CSVStruct.Records = row
	someBytes, err = json.Marshal(u.CSVStruct)

	return someBytes, nil
}

// getRawRowFromJSON parses out a single row from a single JSON message
func (u *httppostwrapper) getRawRowFromJSON(byteValue []byte) (someBytes []byte, err error) {

	var result map[string]interface{}
	err = json.Unmarshal([]byte(byteValue), &result)
	if err != nil {
		return someBytes, fmt.Errorf("can not unmarshal json input file %v", err)
	}

	jsonStruct := extractapi.IntermediateFormat{
		Path:        u.DP.Path,
		Dataprov:    u.DP.ID,
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Messages:    make([]map[string]interface{}, 0),
	}
	jsonStruct.Messages = append(jsonStruct.Messages, result)

	someBytes, err = json.Marshal(jsonStruct)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error Marshaling raw json")
	}

	return someBytes, err
}
