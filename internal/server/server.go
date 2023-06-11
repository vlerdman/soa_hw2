package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"soa_hw_2/internal/pb"
	"sync"
)

type PlayerInfo struct {
	username string
	session  *Session
}

type MafiaServer struct {
	pb.UnimplementedMafiaServer

	idToPlayerInfo map[uuid.UUID]*PlayerInfo
	lastSession    *Session
	mutex          sync.Mutex
}

func NewMafiaServer() *MafiaServer {
	return &MafiaServer{idToPlayerInfo: make(map[uuid.UUID]*PlayerInfo), mutex: sync.Mutex{}}
}

func (ms *MafiaServer) StartSession(req *pb.StartSessionRequest, s pb.Mafia_StartSessionServer) error {
	ms.mutex.Lock()

	if ms.lastSession == nil {
		ms.lastSession = NewSession()
	}

	id := uuid.New()
	ms.idToPlayerInfo[id] = &PlayerInfo{req.Username, ms.lastSession}

	events := make(chan pb.SessionEvent, 10)
	err := ms.lastSession.AddPlayer(req.Username, events)

	if err != nil {
		ms.mutex.Unlock()
		return err
	}

	session := ms.lastSession

	if ms.lastSession.isStarted {
		ms.lastSession = nil
	}
	ms.mutex.Unlock()

	err = s.SendHeader(pb.WithSessionID(id))

	if err != nil {
		session.RemovePlayer(req.Username)
		return err
	}

	for {
		select {
		case event := <-events:
			err := s.Send(&event)
			if err != nil {
				session.RemovePlayer(req.Username)
				return err
			}
		case <-s.Context().Done():
			session.RemovePlayer(req.Username)
			return nil
		}

	}

	return nil
}

func (ms *MafiaServer) Vote(ctx context.Context, req *pb.VoteRequest) (*pb.Empty, error) {
	playerInfo, err := ms.getPlayerInfo(ctx)
	if err != nil {
		return nil, err
	}
	err = playerInfo.session.Vote(playerInfo.username, req.Username)
	return &pb.Empty{}, err
}

func (ms *MafiaServer) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	playerInfo, err := ms.getPlayerInfo(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := playerInfo.session.Check(playerInfo.username, req.Username)
	return resp, err
}

func (ms *MafiaServer) GetSessionState(ctx context.Context, req *pb.Empty) (*pb.SessionState, error) {
	playerInfo, err := ms.getPlayerInfo(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := playerInfo.session.GetState(playerInfo.username)
	return resp, err
}

func (ms *MafiaServer) getPlayerInfo(ctx context.Context) (*PlayerInfo, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("metadata is not provided")
	}
	id, err := pb.FetchSessionID(md)
	if err != nil {
		return nil, err
	}
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	info, ok := ms.idToPlayerInfo[id]
	if !ok {
		return nil, fmt.Errorf("invalid id is provided")
	}
	return info, nil
}
