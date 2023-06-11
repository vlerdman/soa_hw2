package server

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"soa_hw_2/internal/pb"
	"sync"
)

const MafiaCount = 1
const SheriffCount = 1
const CivilianCount = 2

type Player struct {
	role     pb.Role
	username string
	liveness bool
	ch       chan pb.SessionEvent
}

type Session struct {
	id         uuid.UUID
	players    map[string]*Player
	isStarted  bool
	isEnded    bool
	mutex      sync.Mutex
	state      int
	votes      map[string]string
	isVoted    bool
	isChecked  bool
	winnerTeam pb.Team
}

type VoteShootInfo struct {
	username string
	chosen   string
}

func NewSession() *Session {
	return &Session{uuid.New(), make(map[string]*Player), false, false, sync.Mutex{}, 0, make(map[string]string), false, false, pb.Team_UNKNOWN_TEAM}
}

func (s *Session) AddPlayer(username string, ch chan pb.SessionEvent) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.isEnded || s.isStarted {
		return fmt.Errorf("No new players allowed to session")
	}

	mafiaCount := 0
	sheriffCount := 0
	civilianCount := 0

	if username == "" {
		return fmt.Errorf("Username is empty")
	}

	roles := []pb.Role{}

	for _, player := range s.players {
		if player.username == username {
			return fmt.Errorf("Username %s is registered in game yet", username)
		}
		if player.role == pb.Role_MAFIA_ROLE {
			mafiaCount++
		} else if player.role == pb.Role_SHERIFF {
			sheriffCount++
		} else {
			civilianCount++
		}
	}

	for i := mafiaCount; i < MafiaCount; i++ {
		roles = append(roles, pb.Role_MAFIA_ROLE)
	}

	for i := sheriffCount; i < SheriffCount; i++ {
		roles = append(roles, pb.Role_SHERIFF)
	}

	for i := civilianCount; i < CivilianCount; i++ {
		roles = append(roles, pb.Role_CIVILIAN)
	}

	role := roles[rand.Intn(len(roles))]

	s.players[username] = &Player{role, username, true, ch}

	joinInfo := pb.SessionEvent_PlayerJoinInfo{Username: username}
	joinEvent := pb.SessionEvent_JoinInfo{
		&joinInfo,
	}
	s.SendEvent(pb.SessionEvent{EventInfo: &joinEvent})
	log.Printf("roles: %d, id: %s", len(roles), s.id)
	if len(roles) == 1 {
		log.Printf("game with id: %s started", s.id)
		s.isStarted = true
		s.state = 1
		for _, player := range s.players {
			state, _ := s.GetStateUnlocked(player.username)
			event := pb.SessionEvent_SessionStartInfo{
				Role:    state.Player.Role,
				Players: state.Players,
			}
			info := pb.SessionEvent_StartInfo{StartInfo: &event}
			player.ch <- pb.SessionEvent{EventInfo: &info}
		}

	}
	return nil
}

func (s *Session) RemovePlayer(username string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	err := s.ValidateState()
	if err != nil {
		return
	}

	player, ok := s.players[username]
	if ok && player.liveness {
		player.liveness = false

		log.Printf("player %s left session", username)

		leftInfo := pb.SessionEvent_PlayerLeftInfo{Username: username}
		leftEvent := pb.SessionEvent_LeftInfo{
			&leftInfo,
		}
		s.SendEvent(pb.SessionEvent{EventInfo: &leftEvent})

		s.UpdateState()
	}
}

func (s *Session) Vote(username string, voted string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Printf("Vote from %s to %s", username, voted)
	err := s.ValidateState()
	if err != nil {
		return err
	}
	if s.state == 0 {
		return fmt.Errorf("no votes in first day")
	}

	votedPlayer, ok := s.players[voted]
	if !ok || !votedPlayer.liveness {
		return fmt.Errorf("invalid voted")
	}

	player, ok := s.players[username]

	if !ok || !player.liveness {
		return fmt.Errorf("invalid player")
	}

	if s.state%2 == 1 {
		if player.role != pb.Role_MAFIA_ROLE {
			return fmt.Errorf("only mafia allowed to vote")
		}
		_, ok := s.votes[username]
		if ok {
			return fmt.Errorf("already voted")
		}
		s.votes[username] = voted
	} else {
		_, ok := s.votes[username]
		if ok {
			return fmt.Errorf("already voted")
		}
		s.votes[username] = voted
	}
	s.UpdateState()
	return nil
}

func (s *Session) GetCounts() (int, int, int, []*Player) {
	alive := []*Player{}
	mafiaCount := 0
	civilianCount := 0
	sheriffCount := 0
	for _, player := range s.players {
		if player.liveness {
			alive = append(alive, player)
			if player.role == pb.Role_MAFIA_ROLE {
				mafiaCount++
			} else if player.role == pb.Role_SHERIFF {
				sheriffCount++
			} else {
				civilianCount++
			}
		}
	}
	return mafiaCount, civilianCount, sheriffCount, alive
}

func (s *Session) UpdateState() {
	err := s.ValidateState()
	if err != nil {
		return
	}

	mafiaCount, civilianCount, sheriffCount, alive := s.GetCounts()

	needed := mafiaCount
	if s.state%2 == 0 {
		needed = len(alive)
	}
	if sheriffCount == 0 {
		s.isChecked = true
	}
	if !s.isChecked && s.state%2 == 1 {
		return
	}
	log.Printf("voted: %d, needed: %d", len(s.votes), needed)
	if len(s.votes) > 0 && len(s.votes) == needed && (s.isChecked || s.state%2 == 0) {
		max, voted := 0, ""
		for _, player := range alive {
			cur := 0
			for _, result := range s.votes {
				if player.username == result {
					cur++
				}
			}
			if cur > max {
				max, voted = cur, player.username
			}
		}

		s.players[voted].liveness = false
		s.votes = make(map[string]string)

		s.state++

		log.Printf("state changed: %d", s.state)

		voteInfo := pb.SessionEvent_VoteInfo{Username: voted}
		voteEvent := pb.SessionEvent_VoteInfo_{
			&voteInfo,
		}
		s.SendEvent(pb.SessionEvent{EventInfo: &voteEvent})
	} else {
		return
	}

	if sheriffCount != 0 {
		s.isChecked = false
	}

	mafiaCount, civilianCount, sheriffCount, _ = s.GetCounts()

	if mafiaCount == 0 {
		log.Printf("game is ended: civilians wins")
		players := s.GetAllPlayers()
		event := pb.SessionEvent_SessionFinishInfo{
			Winners: pb.Team_CIVILIANS,
			Players: players,
		}
		info := pb.SessionEvent_FinishInfo{FinishInfo: &event}

		s.SendEvent(pb.SessionEvent{EventInfo: &info})
		s.isEnded = true
	}

	if mafiaCount == civilianCount+sheriffCount {
		log.Printf("game is ended: mafia wins")
		players := s.GetAllPlayers()
		event := pb.SessionEvent_SessionFinishInfo{
			Winners: pb.Team_MAFIA,
			Players: players,
		}
		info := pb.SessionEvent_FinishInfo{FinishInfo: &event}

		s.SendEvent(pb.SessionEvent{EventInfo: &info})
		s.isEnded = true
	}

}

func (s *Session) ValidateState() error {
	if !s.isStarted {
		return fmt.Errorf("session is not started")
	}
	if s.isEnded {
		return fmt.Errorf("session is ended")
	}
	return nil
}

func (s *Session) Check(username string, checked string) (*pb.CheckResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.Printf("Check from %s to %s", username, checked)

	err := s.ValidateState()
	if err != nil {
		log.Printf("error: %s", err)
		return nil, err
	}

	player, _ := s.players[username]

	if !player.liveness || player.role != pb.Role_SHERIFF || s.isChecked {
		err = fmt.Errorf("player can't check")
		log.Printf("error: %s", err)
		return nil, err
	}

	if username == checked {
		err = fmt.Errorf("player can't check himself")
		log.Printf("error: %s", err)
		return nil, err
	}

	checkedPlayer, ok := s.players[checked]
	if !ok || !checkedPlayer.liveness {
		err = fmt.Errorf("player can't check himself")
		log.Printf("error: %s", err)
		return nil, err
	}

	s.isChecked = true
	s.UpdateState()
	return &pb.CheckResponse{Username: checked, Role: checkedPlayer.role}, nil
}

func (s *Session) GetState(username string) (*pb.SessionState, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	player, ok := s.players[username]
	if !ok {
		return nil, fmt.Errorf("invalid player")
	}

	protoPlayer := &pb.Player{
		Role:     player.role,
		Username: player.username,
		Liveness: player.liveness,
	}
	protoPlayers := []*pb.Player{}

	for _, p := range s.players {
		prPl := &pb.Player{
			Role:     p.role,
			Username: p.username,
			Liveness: p.liveness,
		}
		if prPl.Username != username {
			prPl.Role = pb.Role_UNKNOWN_ROLE
		}
		protoPlayers = append(protoPlayers, prPl)
	}

	state := pb.SessionState{
		Player:     protoPlayer,
		Players:    protoPlayers,
		WinnerTeam: s.winnerTeam,
	}
	return &state, nil
}

func (s *Session) GetStateUnlocked(username string) (*pb.SessionState, error) {
	player, ok := s.players[username]
	if !ok {
		return nil, fmt.Errorf("invalid player")
	}

	protoPlayer := &pb.Player{
		Role:     player.role,
		Username: player.username,
		Liveness: player.liveness,
	}
	protoPlayers := []*pb.Player{}

	for _, p := range s.players {
		prPl := &pb.Player{
			Role:     p.role,
			Username: p.username,
			Liveness: p.liveness,
		}
		if prPl.Username != username {
			prPl.Role = pb.Role_UNKNOWN_ROLE
		}
		protoPlayers = append(protoPlayers, prPl)
	}

	state := pb.SessionState{
		Player:     protoPlayer,
		Players:    protoPlayers,
		WinnerTeam: s.winnerTeam,
	}
	return &state, nil
}

func (s *Session) GetAllPlayers() []*pb.Player {
	protoPlayers := []*pb.Player{}

	for _, p := range s.players {
		prPl := &pb.Player{
			Role:     p.role,
			Username: p.username,
			Liveness: p.liveness,
		}
		protoPlayers = append(protoPlayers, prPl)
	}
	return protoPlayers
}

func (s *Session) SendEvent(event pb.SessionEvent) {
	for _, p := range s.players {
		p.ch <- event
	}
}
