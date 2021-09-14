package main

import (
	"context"

	"github.com/churrodata/churro/internal/ctl"
	pb "github.com/churrodata/churro/rpc/ctl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"

	"net"
	"os"
	"testing"
	"time"
)

func Server() {
	zerolog.TimeFieldFormat = time.RFC822

	log.Logger = log.With().Caller().Logger()

	lis, err := net.Listen("tcp", ctl.DefaultPort)
	if err != nil {
		log.Error().Stack().Err(err).Msg("failed to listen ")
		os.Exit(1)
	}
	s := grpc.NewServer()
	server := ctl.Server{}
	pb.RegisterCtlServer(s, &server)
	if err := s.Serve(lis); err != nil {
		log.Error().Stack().Err(err).Msg("failed to serve")
		os.Exit(1)
	}
}

func TestMain(m *testing.M) {
	go Server()
	time.Sleep(2 * time.Second)
	os.Exit(m.Run())
}

func TestClient(t *testing.T) {
	conn, err := grpc.Dial(ctl.DefaultPort, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewCtlClient(conn)

	t.Run("Ping", func(t *testing.T) {
		_, err := c.Ping(context.Background(), &pb.PingRequest{})
		if err != nil {
			t.Fatalf("could not ping: %v", err)
		}

	})

}
