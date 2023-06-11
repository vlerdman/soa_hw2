package client

import (
	"context"
	"fmt"
	"log"
	"soa_hw_2/internal/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	ctx    context.Context
	cli    pb.MafiaClient
	stream pb.Mafia_StartSessionClient
	events chan *pb.SessionEvent
}

func NewClient(ctx context.Context, username string, conn *grpc.ClientConn) (*Client, error) {
	cli := pb.NewMafiaClient(conn)

	stream, err := cli.StartSession(ctx, &pb.StartSessionRequest{Username: username})
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %s", err)
	}

	md, err := stream.Header()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %s", err)
	}

	sessionID, err := pb.FetchSessionID(md)
	if err != nil {
		return nil, fmt.Errorf("failed to get id: %s", err)
	}

	ctx = metadata.NewOutgoingContext(ctx, pb.WithSessionID(sessionID))

	return &Client{
		ctx:    ctx,
		cli:    cli,
		stream: stream,
		events: make(chan *pb.SessionEvent),
	}, nil
}

func (c *Client) Events() <-chan *pb.SessionEvent {
	return c.events
}

func (c *Client) ForwardEvents() {
	for {
		event, err := c.stream.Recv()
		if err != nil {
			log.Fatalf("\n\nServer closed\n")
			return
		}

		c.events <- event
	}
}

func (c *Client) Vote(username string) error {
	_, err := c.cli.Vote(c.ctx, &pb.VoteRequest{Username: username})

	return err
}

func (c *Client) Check(username string) (string, error) {
	resp, err := c.cli.Check(c.ctx, &pb.CheckRequest{Username: username})
	if err != nil {
		return "", err
	}

	return RoleToString(resp.Role), err
}

func (c *Client) GetState() (*pb.SessionState, error) {
	return c.cli.GetSessionState(c.ctx, &pb.Empty{})
}
