package main

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/churrodata/churro/internal/extractsource"
	pb "github.com/churrodata/churro/rpc/extractsource"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
)

func Server() {

	zerolog.TimeFieldFormat = time.RFC822

	log.Logger = log.With().Caller().Logger()
	lis, err := net.Listen("tcp", extractsource.DefaultPort)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		os.Exit(1)
	}
	s := grpc.NewServer()
	server := extractsource.Server{}
	pb.RegisterExtractSourceServer(s, &server)
	if err := s.Serve(lis); err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		os.Exit(1)
	}
}

func TestMain(m *testing.M) {
	go Server()
	time.Sleep(2 * time.Second)
	os.Exit(m.Run())
}

func TestClient(t *testing.T) {
	log.Info().Msg("churro-extractsource-test TestClient")

	conn, err := grpc.Dial(extractsource.DefaultPort, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewExtractSourceClient(conn)

	t.Run("Ping", func(t *testing.T) {
		_, err := c.Ping(context.Background(), &pb.PingRequest{})
		if err != nil {
			t.Fatalf("could not ping: %v", err)
		}

	})

}
