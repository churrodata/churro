// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"

	"aqwari.net/xml/xmltree"
	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/santhosh-tekuri/jsonschema/v3"
)

// UploadSchema ...
func (u *HandlerWrapper) UploadSchema(w http.ResponseWriter, r *http.Request) {

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Error Retrieving the File")
		w.Write([]byte(err.Error()))
		return
	}
	defer file.Close()

	log.Info().Msg(fmt.Sprintf("Uploaded File: %+v", handler.Filename))
	log.Info().Msg(fmt.Sprintf("File Size: %+v", handler.Size))
	log.Info().Msg(fmt.Sprintf("MIME Header: %+v", handler.Header))

	vars := mux.Vars(r)
	pipelineID := vars["id"]
	extractSourceID := vars["wdid"]

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pipelineClient, err := pkg.NewClient(config, "")
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	pList, err := pipelineClient.List()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		w.Write([]byte(err.Error()))
		return
	}

	var x v1alpha1.Pipeline
	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			x = pList.Items[i]
		}
	}

	// get the extract source dir so we know the scheme we are dealing with
	client, err := GetServiceConnection(x.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in uploadschema b")
		w.Write([]byte(err.Error()))
		return
	}

	req := pb.GetExtractSourceRequest{
		Namespace:       x.Name,
		ExtractSourceID: extractSourceID,
	}

	response, err := client.GetExtractSource(context.Background(), &req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	var value domain.ExtractSource
	err = json.Unmarshal([]byte(response.ExtractSourceString), &value)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in uploadschema c ")
		w.Write([]byte(err.Error()))
		return
	}

	log.Info().Msg("extract source scheme is " + value.Scheme)

	if value.Initialized {
		x := HandlerWrapper{}
		x.DatabaseType = u.DatabaseType
		x.ErrorText = "extract source directory already initialized"
		x.StatusText = "extract source dir initialized"
		x.PipelineExtractSource(w, r)
		return
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 8096)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in uploadschema e ")
			a := u.Copy(err.Error())
			a.PipelineExtractSource(w, r)
			return
		}

		log.Info().Msg(fmt.Sprintf("bytes read %d\n", n))
	}

	value.ExtractRules = make(map[string]domain.ExtractRule)

	if value.Scheme == extractapi.JSONPathScheme {
		err := handleJSONPath(buffer, value.ExtractRules)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in jsonpath uploadschema ff ")
			x := HandlerWrapper{}
			x.DatabaseType = u.DatabaseType
			x.ErrorText = err.Error()
			x.StatusText = ""
			x.PipelineExtractSource(w, r)
			return
		}
	} else if value.Scheme == extractapi.CSVScheme {

		err := handleCSV(buffer, value.ExtractRules)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in csv uploadschema ff ")
			x := HandlerWrapper{}
			x.DatabaseType = u.DatabaseType
			x.ErrorText = err.Error()
			x.StatusText = ""
			x.PipelineExtractSource(w, r)
			return
		}

	} else if value.Scheme == extractapi.XMLScheme {
		err := handleXMLPath(buffer, value.ExtractRules)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error in xsd uploadschema ")
			x := HandlerWrapper{}
			x.DatabaseType = u.DatabaseType
			x.ErrorText = err.Error()
			x.StatusText = ""
			x.PipelineExtractSource(w, r)
			return
		}
	} else {
		log.Error().Stack().Msg("error in uploadschema h ")
		x := HandlerWrapper{}
		x.DatabaseType = u.DatabaseType
		x.ErrorText = "scheme not supported yet"
		x.StatusText = ""
		x.PipelineExtractSource(w, r)
		return
	}

	// if extract rules were found, then write them to the db
	if len(value.ExtractRules) > 0 {
		for _, v := range value.ExtractRules {
			v.ExtractSourceID = extractSourceID
			b, _ := json.Marshal(v)
			ereq := pb.CreateExtractRuleRequest{
				ExtractRuleString: string(b),
				Namespace:         x.Name,
			}

			_, err := client.CreateExtractRule(context.Background(), &ereq)
			if err != nil {
				x := HandlerWrapper{}
				x.DatabaseType = u.DatabaseType
				x.ErrorText = err.Error()
				x.StatusText = ""
				x.PipelineExtractSource(w, r)
				return
			}
		}
	}

	j := HandlerWrapper{}
	j.DatabaseType = u.DatabaseType
	j.ErrorText = ""
	j.StatusText = "schema uploaded successfully"
	j.PipelineExtractSource(w, r)
	return
}

func handleCSV(schemaBytes []byte, rulesMap map[string]domain.ExtractRule) error {

	// read the csv file, get the header
	someReader := bytes.NewReader(schemaBytes)

	csvReader := csv.NewReader(someReader)

	//read the 1st row which we assume is a header
	record, err := csvReader.Read()
	if err == io.EOF {
		log.Error().Stack().Err(err).Msg("error in uploadschema ff ")
		return err
	}
	if err != nil {
		log.Error().Stack().Err(err).Msg("error in uploadschema f ")
		return err
	}
	// process the csv header which we expect to be there
	log.Info().Msg(fmt.Sprintf("%+v\n", record))
	for i := 0; i < len(record); i++ {
		log.Info().Msg("col " + record[0])
		er := domain.ExtractRule{
			ID:         xid.New().String(),
			ColumnName: record[i],
			ColumnPath: strconv.Itoa(i),
			ColumnType: extractapi.COLTYPE_TEXT,
		}
		rulesMap[er.ID] = er
	}
	return nil
}

func handleJSONPath(schemaBytes []byte, rulesMap map[string]domain.ExtractRule) error {

	file, err := ioutil.TempFile("/tmp", "jsonpathschema.*.json")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error createing tempfile ")
		return err
	}
	defer os.Remove(file.Name())

	bytesWritten, err := file.Write(schemaBytes)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error writing tempfile ")
		return err
	}

	log.Info().Msg(fmt.Sprintf("wrote jsonpath schema temp file %d bytes\n", bytesWritten))
	compiler := jsonschema.NewCompiler()

	schema, err := compiler.Compile(file.Name())
	if err != nil {
		log.Error().Stack().Err(err).Msg("error compiling json path schema ")
		return err
	}

	GetRulesFromJSONSchema("$", rulesMap, schema)

	return nil
}

// GetRulesFromJSONSchema ...
func GetRulesFromJSONSchema(path string, rules map[string]domain.ExtractRule, schema *jsonschema.Schema) {
	props := schema.Properties
	log.Info().Msg(fmt.Sprintf("properties len %d\n", len(props)))

	for k, v := range props {
		//log.Info().Msg(fmt.Sprintf("key %s %+v\n", k, v.Types))
		log.Info().Msg("key  " + k)
		for i := 0; i < len(v.Types); i++ {
			if v.Types[i] == "array" {
				log.Info().Msg("type is an array and needs work")
				log.Info().Msg(fmt.Sprintf("and its items are %+v\n", v.Items))
				if w, ok := v.Items.([]*jsonschema.Schema); ok {
					//var s []*jsonschema.Schema
					// *jsonschema.Schema, not []*jsonschema.Schema
					//s = v.Items.([]*jsonschema.Schema)
					log.Info().Msg(fmt.Sprintf("and its itemss len are %d\n", len(w)))
					for x := 0; x < len(w); x++ {
						pathfoo := path + "." + k
						//printProps(pathfoo, rules, v)
						GetRulesFromJSONSchema(pathfoo, rules, w[x])
					}
				} else if w, ok := v.Items.(*jsonschema.Schema); ok {
					//var s *jsonschema.Schema
					// *jsonschema.Schema, not []*jsonschema.Schema
					//s = v.Items.(*jsonschema.Schema)
					pathfoo := path + "." + k
					//printProps(pathfoo, rules, v)
					GetRulesFromJSONSchema(pathfoo, rules, w)
				}
				//printProps(pathfoo, rules, s)
			} else if v.Types[i] == "object" {
				log.Info().Msg("type is an object and needs work")
				log.Info().Msg(fmt.Sprintf("and its props are %+v\n", v.Properties))
				pathfoo := path + "." + k
				GetRulesFromJSONSchema(pathfoo, rules, v)
			} else {
				log.Info().Msg(fmt.Sprintf("type is %s\n", v.Types[i]))
				r := domain.ExtractRule{
					ID:         xid.New().String(),
					ColumnName: k,
					ColumnType: v.Types[i],
					ColumnPath: path + "." + k,
				}
				rules[r.ID] = r
			}
		}
	}
	//log.Info().Msg(fmt.Sprintf("schema +%v\n", schema))

}

func handleXMLPath(schemaBytes []byte, rulesMap map[string]domain.ExtractRule) error {

	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return err
	}
	doc, _ := xmltree.Parse(schemaBytes)

	xsdWalk(rulesMap, reg, "", "", doc)

	return nil
}

func xsdWalk(rulesMap map[string]domain.ExtractRule, reg *regexp.Regexp, fullPath string, prefix string, el *xmltree.Element) {
	scope := join(prefix, el.Name.Local)
	//println(scope)
	if el.Name.Local == "element" {
		//log.Info().Msg(fmt.Sprintf("%+v\n", el.StartElement.Attr))
		for i := 0; i < len(el.StartElement.Attr); i++ {
			em := el.StartElement.Attr[i]
			if em.Name.Local == "name" {

				if len(el.Children) == 0 {
					//log.Info().Msg(fmt.Sprintf("full=%s children=%d name=[%s] value[%s] full=[%s]\n", fullPath, len(el.Children), em.Name.Local, em.Value, fullPath+"/"+em.Value))
					r := domain.ExtractRule{
						ColumnPath: fullPath + "/" + em.Value,
					}
					r.ColumnName = reg.ReplaceAllString(r.ColumnPath, "")
					log.Info().Msg(fmt.Sprintf("rulename=%s rule=[%s]\n", r.ColumnName, r.ColumnPath))
					rulesMap[r.ColumnName] = r
				}
				fullPath = fullPath + "/" + em.Value
			}

		}
	}
	for _, c := range el.Children {
		xsdWalk(rulesMap, reg, fullPath, scope, &c)
	}
}

func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
