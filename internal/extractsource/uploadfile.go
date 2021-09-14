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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg"
	pb "github.com/churrodata/churro/rpc/extractsource"
	rpcwatch "github.com/churrodata/churro/rpc/extractsource"
	"github.com/rs/zerolog/log"
)

// UploadToExtractSource ...
func (s *Server) UploadToExtractSource(stream rpcwatch.ExtractSource_UploadToExtractSourceServer) error {
	req, err := stream.Recv()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	extractSourceID := req.GetInfo().GetExtractSourceID()
	fileName := req.GetInfo().GetFileName()

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting kubeclientset err ")
		return err
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}
	wd := domain.ExtractSource{}

	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		if extractSourceID == pipelineToUpdate.Spec.Extractsources[i].ID {
			wd.Name = pipelineToUpdate.Spec.Extractsources[i].Name
			wd.Path = pipelineToUpdate.Spec.Extractsources[i].Path

		}
	}

	fileData := bytes.Buffer{}
	fileSize := 0

	for {
		err := contextError(stream.Context())
		if err != nil {
			log.Error().Stack().Err(err).Msg("some error")
			return err
		}
		log.Info().Msg("waiting to receive more data")

		req, err := stream.Recv()
		if err == io.EOF {
			log.Error().Stack().Err(err).Msg("no more file data")
			break
		}
		if err != nil {
			log.Error().Stack().Err(err).Msg("stream recv error")
			return err
		}
		chunk := req.GetChunkData()
		size := len(chunk)
		log.Info().Msg(fmt.Sprintf("received chunk size %d\n", size))

		fileSize += size
		_, err = fileData.Write(chunk)
		if err != nil {
			log.Error().Stack().Err(err).Msg("file write error")
			return err
		}
	}

	finalpath := fmt.Sprintf("/%s/%s", wd.Path, fileName)
	readypath := fmt.Sprintf("/%s/ready/%s", wd.Path, fileName)
	err = ioutil.WriteFile(readypath, fileData.Bytes(), 0644)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	// example move /csvfiles/ready/source.csv to /csvfiles/source.csv
	// move it from ready subdir up a level that is being watched
	err = RenameFile(readypath, finalpath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("rename error")
		return err
	}

	res := &pb.UploadToExtractSourceResponse{
		FilePath: finalpath,
		FileSize: uint32(fileSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		log.Error().Stack().Err(err).Msg("sendandclose error")
		return err
	}
	return nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return errors.New("request is canceled")
	case context.DeadlineExceeded:
		return errors.New("deadline is exceeded")
	default:
		return nil
	}
}
