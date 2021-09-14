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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/extractsource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadByURL ...
func (s *Server) UploadByURL(ctx context.Context, req *pb.UploadByURLRequest) (response *pb.UploadByURLResponse, err error) {

	resp := &pb.UploadByURLResponse{}

	log.Info().Msg(fmt.Sprintf("UploadByURL %+v\n", req))

	if req.FileURL == "" {
		log.Error().Stack().Msg("error fileURL is empty")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	namespace := os.Getenv("CHURRO_NAMESPACE")
	if namespace == "" {
		log.Error().Stack().Err(err).Msg("error CHURRO_NAMESPACE is empty")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting kubeclientset err ")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	wd := domain.ExtractSource{}
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if req.ExtractSourceID == pipelineToUpdate.Spec.Extractsources[i].ID {
			wd.Name = pipelineToUpdate.Spec.Extractsources[i].Name
			wd.Path = pipelineToUpdate.Spec.Extractsources[i].Path

		}
	}
	err = downloadFile(wd.Path, req.FileURL)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error downloading file " + req.FileURL)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return resp, nil
}

func downloadFile(wdirPath, fullURLFile string) error {

	var fileName string

	// Build fileName from fullPath
	fileURL, err := url.Parse(fullURLFile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	path := fileURL.Path
	segments := strings.Split(path, "/")
	shortName := segments[len(segments)-1]
	fileName = wdirPath + "/" + shortName
	log.Info().Msg("download path is " + fileName)

	// Create blank file
	file, err := os.Create(fileName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{
		Transport: tr,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	// Put content on file
	resp, err := client.Get(fullURLFile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	defer file.Close()
	log.Info().Msg(fmt.Sprintf("Downloaded a file %s with size %d", fileName, size))

	// move /tmp/foo to /wdirpath/foo
	readyName := wdirPath + "/ready/" + shortName
	targetName := wdirPath + "/" + shortName
	err = RenameFile(readyName, targetName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	log.Info().Msg(fmt.Sprintf("Moved file %s to %s", readyName, targetName))

	return nil

}

// RenameFile ...
func RenameFile(sourcePath, destPath string) error {
	log.Info().Msg(fmt.Sprintf("renaming from [%s] to [%s]\n", sourcePath, destPath))
	err := os.Rename(sourcePath, destPath)
	if err != nil {
		return fmt.Errorf("Couldn't rename file: %s", err)
	}
	return nil
}

// MoveFile ...
func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}
