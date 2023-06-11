package main

import (
	"fmt"
	"log"
	"net"
	"soa_hw_2/internal/pb"
	"soa_hw_2/internal/server"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	DefaultTimeout   = 10 * time.Second
	DefaultKeepAlive = 500 * time.Millisecond
)

func main() {
	address := "0.0.0.0:9000"
	srv, lis, err := registerServer(address)
	if err != nil {
		log.Fatalf("server registration failed on %s: %v\n", address, err)
	}

	err = srv.Serve(lis)
	if err != nil {
		log.Fatalf("server failed: %v\n", err)
	}

}

func registerServer(address string) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("can't start listen")
	}

	grpcServer := grpc.NewServer(
		grpc.ConnectionTimeout(DefaultTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: DefaultKeepAlive,
			Time:              DefaultKeepAlive,
			Timeout:           DefaultKeepAlive,
		}),
	)

	pb.RegisterMafiaServer(grpcServer, server.NewMafiaServer())

	return grpcServer, lis, nil
}
