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
	"context"
	"errors"
	"fmt"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/pkg"
	"github.com/rs/zerolog/log"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetStorageClasses ...
func GetStorageClasses() (names []string, err error) {
	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return names, err
	}

	var storageClasses *storagev1.StorageClassList
	storageClasses, err = clientset.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return names, err
	}
	for _, sc := range storageClasses.Items {
		names = append(names, sc.Name)
	}

	return names, err
}

func GetSupportedDatabases() (names []string, err error) {
	result, err := pkg.GetChurroui()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting churroui")
		return names, err
	}
	log.Info().Msg(fmt.Sprintf("got ui CR %+v\n", result))

	return result.Spec.Supporteddatabases, nil
}

func getPipelineCR(pipelineID string) (x v1alpha1.Pipeline, err error) {
	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting Pipeline CR")
		return x, err
	}

	pipelineClient, err := pkg.NewClient(config, "")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting Pipeline CR")
		return x, err
	}

	pList, err := pipelineClient.List()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting Pipeline CR")
		return x, err
	}

	for i := 0; i < len(pList.Items); i++ {
		if pipelineID == pList.Items[i].Spec.Id {
			return pList.Items[i], nil
		}
	}

	return x, errors.New("pipeline CR " + pipelineID + " not found")

}
