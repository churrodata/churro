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
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/pkg"
	"github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/zerolog/log"

	pb "github.com/churrodata/churro/rpc/ctl"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MetricFiles is the key name
const MetricFiles = "Files Processed"

// GetPipelineStatus gets information on a pipeline
func (s *Server) GetPipelineStatus(ctx context.Context, req *ctl.GetPipelineStatusRequest) (*pb.GetPipelineStatusResponse, error) {

	mymetrics, err := s.getMetrics()
	if err != nil {
	}

	myjobs, err := s.getJobs()
	if err != nil {
	}

	resp := &pb.GetPipelineStatusResponse{
		Metrics: mymetrics,
		Jobs:    myjobs,
	}

	return resp, nil
}

func (s Server) getMetrics() (metrics []*pb.PipelineMetric, err error) {

	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return metrics, err
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.AdminDataSource)
	if err != nil {
		return metrics, err
	}

	m, err := churroDB.GetAllPipelineMetrics()
	if err != nil {
		return metrics, err
	}

	for i := 0; i < len(m); i++ {
		m1 := pb.PipelineMetric{
			Name:  m[i].Name,
			Value: m[i].Value,
		}
		metrics = append(metrics, &m1)
	}

	return metrics, nil
}

func (s Server) getJobs() (jobs []*pb.PipelineJobStatus, err error) {
	var churroDB db.ChurroDatabase
	churroDB, err = db.NewChurroDB(s.Pi.Spec.DatabaseType)
	if err != nil {
		return jobs, err
	}

	err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.DataSource)
	if err != nil {
		return jobs, err
	}

	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return jobs, err
	}

	jobs = make([]*ctl.PipelineJobStatus, 0)

	labelSelector := fmt.Sprintf("service=churro-extract")
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	namespace := s.Pi.Name
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return jobs, err
	}
	for i := 0; i < len(pods.Items); i++ {
		p := pods.Items[i]
		m1 := pb.PipelineJobStatus{
			Name:          p.Name,
			Datasource:    p.ObjectMeta.Labels["watchdirname"],
			Status:        string(p.Status.Phase),
			StartDate:     p.Status.StartTime.Format("2006-01-02 15:04:05"),
			CompletedDate: "",
		}
		//	if p.Status.Phase == v1.PodSucceeded {
		for i := 0; i < len(p.Status.ContainerStatuses); i++ {
			status := p.Status.ContainerStatuses[i]
			if status.State.Terminated != nil {
				t := status.State.Terminated.FinishedAt
				m1.CompletedDate = t.Format("2006-01-02 15:04:05")
			}
		}
		//}

		jp, err := churroDB.GetExtractLog(p.Name)
		if err != nil {
			return jobs, err
		}
		m1.RecordsLoaded = int32(jp.RecordsLoaded)
		m1.FileName = jp.FileName
		m1.TableName = jp.TableName
		log.Info().Msg(fmt.Sprintf("adding jobProfile of %s/%s", m1.FileName, m1.TableName))
		jobs = append(jobs, &m1)
	}

	return jobs, nil
}

func getPodLog(ctx context.Context, client kubernetes.Interface, pod *v1.Pod) ([]byte, error) {
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	return buf.Bytes(), err
}

// GetPipelineJobLog fetches a extract job log
func (s *Server) GetPipelineJobLog(ctx context.Context, req *ctl.GetPipelineJobLogRequest) (*pb.GetPipelineJobLogResponse, error) {

	log.Info().Msg(fmt.Sprintf("job log request %+v\n", req))

	resp := &pb.GetPipelineJobLogResponse{}

	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return resp, err
	}
	pod := &v1.Pod{}
	pod.Name = req.Podname
	pod.Namespace = req.Namespace
	logBytes, err := getPodLog(context.TODO(), clientset, pod)
	if err != nil {
		return resp, err
	}
	resp.Logstring = string(logBytes)

	return resp, nil
}

// DeleteJobs deletes extract job Pods
func (s *Server) DeleteJobs(ctx context.Context, req *ctl.DeleteJobsRequest) (*pb.DeleteJobsResponse, error) {

	resp := &pb.DeleteJobsResponse{}

	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		return resp, err
	}

	do := metav1.DeleteOptions{}

	for i := 0; i < len(req.Jobs); i++ {
		job := req.Jobs[i]

		err := clientset.CoreV1().Pods(req.Namespace).Delete(ctx, job.Name, do)
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}
