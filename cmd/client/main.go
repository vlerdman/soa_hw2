package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"os/signal"
	"soa_hw_2/internal/client"
	"syscall"
)

func main() {
	address := "dns:///mafiaserver:9000"

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var username string
	fmt.Printf("Enter your username: ")
	_, _ = fmt.Scanf("%s", &username)

	conn, err := createConnection(address)
	if err != nil {
		log.Fatalf("failed to connect to %s: %v\n", address, err)
	}
	defer conn.Close()

	cli, err := client.NewClient(ctx, username, conn)
	if err != nil {
		log.Fatalf("failed to init gRPC client: %v\n", err)
	}

	messenger := client.NewMessenger(ctx)
	handler := client.NewHandler(cli, messenger)

	go messenger.Start()
	go handler.Start()
	go cli.ForwardEvents()

	<-stop
	cancel()
}

func createConnection(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	return conn, err
}
