// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package transform

import (
	"fmt"

	"github.com/churrodata/churro/internal/domain"
	"github.com/rs/zerolog/log"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// RunRules ...
func RunRules(cols []string, record []interface{}, rules map[string]domain.ExtractRule, functions []domain.TransformFunction) error {
	log.Info().Msg(fmt.Sprintf("functions entering RunRules %v", functions))
	log.Info().Msg(fmt.Sprintf("records entering RunRules %v", record))
	for _, rule := range rules {
		if rule.TransformFunction == "" || rule.TransformFunction == "None" {
			continue
		}

		log.Info().Msg(fmt.Sprintf("cols coming into to transform %v\n", cols))
		var recordIndex int
		//var err error
		for i := 0; i < len(cols); i++ {
			if cols[i] == rule.ColumnName {
				log.Info().Msg(fmt.Sprintf("rule column name found in cols list at %d col at this index is %s value is %s\n", i, cols[i], record[i].(string)))
				recordIndex = i
			}
		}
		/**
		recordIndex, err := strconv.Atoi(rule.ColumnPath)
		if err != nil {
			// assume an error means we have a string
			recordIndex, err = findColumn(cols, rule.ColumnPath)
			if err != nil {
				return err
			}
		}
		*/
		var fn domain.TransformFunction
		log.Info().Msg(fmt.Sprintf("looking for function %v\n", rule.TransformFunction))
		fn, err := findFunction(functions, rule.TransformFunction)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}
		i := interp.New(interp.Options{})
		i.Use(stdlib.Symbols)
		log.Info().Msg("function source is " + fn.Source)
		_, err = i.Eval(fn.Source)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return fmt.Errorf("error in interpreting source %v", err)
		}
		log.Info().Msg("function is " + rule.TransformFunction)
		v, err := i.Eval(fn.Name)
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return fmt.Errorf("error in interpreting function %v", err)
		}
		//tmp := v.Interface().(func(string) string)(record[recordIndex].(string))
		tmp := v.Interface().(func(string) string)
		log.Info().Msg(fmt.Sprintf("value to convert is index is %d  value is %s ", recordIndex, record[recordIndex].(string)))
		record[recordIndex] = tmp(record[recordIndex].(string))
	}
	return nil
}

func findColumn(cols []string, path string) (int, error) {
	for i := 0; i < len(cols); i++ {
		if cols[i] == path {
			return i, nil
		}
	}
	return 0, fmt.Errorf("could not find column that matches path %s", path)
}

func findFunction(functions []domain.TransformFunction, functionName string) (domain.TransformFunction, error) {
	for i := 0; i < len(functions); i++ {
		if functions[i].Name == functionName {
			return functions[i], nil
		}
	}
	return domain.TransformFunction{}, fmt.Errorf("could not find function %s", functionName)
}
