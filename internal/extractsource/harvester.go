// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package extractsource

import (
	"context"
	"fmt"
	"time"

	"github.com/churrodata/churro/pkg"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultHarvestFreq how often to peform the harvest check
	DefaultHarvestFreq = "@every 20s"
	// DefaultHarvestPodDuration is the age of pods to harvest
	DefaultHarvestPodDuration = "44h"
)

// StartHarvesting ...
func (s *Server) StartHarvesting() {
	c := cron.New()
	// TODO this cron expression could be confgurable via
	// the Pipeline CR eventually
	cronExpression := DefaultHarvestFreq
	if s.Pi.Spec.HarvestFrequency != "" {
		cronExpression = s.Pi.Spec.HarvestFrequency
	}

	c.AddFunc(cronExpression, func() {
		log.Info().Msg("inside cron function")
		s.harvest()
	})

	c.Start()

	log.Info().Msg("harvesting started ")

}

func (s *Server) harvest() {

	label := "service=churro-extract"

	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return
	}

	listOptions := metav1.ListOptions{LabelSelector: label}
	pods, err := clientset.CoreV1().Pods(s.Pi.Name).List(context.TODO(), listOptions)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return
	}
	myDuration := DefaultHarvestPodDuration
	if s.Pi.Spec.HarvestPodDuration != "" {
		myDuration = s.Pi.Spec.HarvestPodDuration
	}
	for i := 0; i < len(pods.Items); i++ {
		p := pods.Items[i]
		switch p.Status.Phase {
		case v1.PodRunning:
		case v1.PodPending:
		default:
			// TODO eventually this duration could be configurable
			// within the Pipeline CR
			myD, err := time.ParseDuration(myDuration)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in parsing time")
				return
			}
			hoursString, difference := durationSinceNow(myD, p.Status.StartTime.Time)
			log.Info().Msg(fmt.Sprintf("harvest info:  pod %s oldAge %s podAge %s ageDiff %f\n", p.Name, myDuration, hoursString, difference))
			if difference > 0 {

				// TODO eventually a hook to deal with archival of logs
				// might go here, that hook could be grpc for example so
				// that customers can write their own pre-delete logic
				// as they wish...

				err = clientset.CoreV1().Pods(s.Pi.Name).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				if err != nil {
					log.Error().Stack().Err(err).Msg("error in deleting pod")
					return
				}
				log.Info().Msg("harvested pod " + p.Name + " due to old age")
			}
		}
	}

}

func durationSinceNow(myDuration time.Duration, startTime time.Time) (string, float64) {
	d := time.Since(startTime)

	if d.Hours() > myDuration.Hours() {
		return d.String() + " hours exceeded", d.Hours() - myDuration.Hours()
	}

	return d.String(), d.Hours() - myDuration.Hours()
}
