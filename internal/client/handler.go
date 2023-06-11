package client

import (
	"fmt"
	"soa_hw_2/internal/pb"
	"strings"
)

type Handler struct {
	client    *Client
	messenger *Messenger
	state     int
}

func NewHandler(client *Client, messenger *Messenger) *Handler {
	return &Handler{
		client:    client,
		messenger: messenger,
		state:     0,
	}
}

func (h *Handler) Start() {
	go h.handleEvents()
	go h.handleInput()
}

func (h *Handler) handleInput() {
	for {
		input := <-h.messenger.input
		switch {
		case strings.HasPrefix(input, "vote"):
			h.vote(input[len("vote")+1:])
		case strings.HasPrefix(input, "check"):
			h.check(input[len("check")+1:])
		case strings.HasPrefix(input, "get_state"):
			h.getState()

		default:
			h.sendOutput("invalid input received")
		}
	}
}

func (h *Handler) vote(username string) {
	err := h.client.Vote(username)
	if err != nil {
		h.sendOutput(fmt.Sprintf("vote error: %s", err))
	}
}

func (h *Handler) check(username string) {
	result, err := h.client.Check(username)
	if err != nil {
		h.sendOutput(fmt.Sprintf("check error: %s", err))
	} else {
		h.sendOutput(fmt.Sprintf("username %s is %s", username, result))
	}
}

func (h *Handler) getState() {
	state, err := h.client.GetState()
	if err != nil {
		h.sendOutput(fmt.Sprintf("get state error: %s", err))
	} else {
		str := "current state:"
		for _, player := range state.Players {
			str += "\n" + PlayerToString(player) + "\n"
		}

		h.sendOutput(str)
	}
}

func (h *Handler) handleEvents() {
	h.handleHelp()
	for {
		event := <-h.client.events

		switch event.EventInfo.(type) {
		case *pb.SessionEvent_JoinInfo:
			h.handleJoin(event.GetJoinInfo())
		case *pb.SessionEvent_StartInfo:
			h.handleStart(event.GetStartInfo())
		case *pb.SessionEvent_VoteInfo_:
			h.handleVote(event.GetVoteInfo())
		case *pb.SessionEvent_LeftInfo:
			h.handleLeft(event.GetLeftInfo())
		case *pb.SessionEvent_FinishInfo:
			h.handleFinish(event.GetFinishInfo())
		default:
			h.sendOutput("invalid event received")
		}
	}
}

func (h *Handler) sendOutput(message string) {
	h.messenger.output <- "\n" + message + "\n"
}

func (h *Handler) handleJoin(info *pb.SessionEvent_PlayerJoinInfo) {
	h.sendOutput(fmt.Sprintf("Player %s join the game", info.Username))
}

func (h *Handler) handleStart(info *pb.SessionEvent_SessionStartInfo) {
	str := fmt.Sprintf("Your role: %s", RoleToString(info.Role))
	str += "\nplayers:\n"
	for _, player := range info.Players {
		str += "\n" + PlayerToString(player) + "\n"
	}
	h.sendOutput(str)
	h.updateState()
}

func (h *Handler) handleLeft(info *pb.SessionEvent_PlayerLeftInfo) {
	h.sendOutput(fmt.Sprintf("Player %s left the game", info.Username))
}

func (h *Handler) handleVote(info *pb.SessionEvent_VoteInfo) {

	if h.state%2 == 1 {
		h.sendOutput(fmt.Sprintf("Player %s was killed by mafia", info.Username))
	} else {
		h.sendOutput(fmt.Sprintf("Player %s was voted", info.Username))
	}

	h.updateState()
}

func (h *Handler) updateState() {
	h.state++
	if h.state%2 == 1 {
		h.sendOutput(fmt.Sprintf("%d night: mafia should vote and sheriff should check", h.state/2+1))
	} else {
		h.sendOutput(fmt.Sprintf("%d day: all should vote", h.state/2+1))
	}

}

func (h *Handler) handleHelp() {
	str := `
usage:
        
    get_state - get current state

    check - check role of player (allowed only for sheriff during night)

    vote - vote for jailing or killing player (jailing allowed only during day, killing - during night for mafia)

`
	h.sendOutput(str)
}

func (h *Handler) handleFinish(info *pb.SessionEvent_SessionFinishInfo) {
	str := fmt.Sprintf("Team %s winned", TeamToString(info.Winners))
	str += "\nplayers:\n"
	for _, player := range info.Players {
		str += "\n" + PlayerToString(player) + "\n"
	}
	h.sendOutput(str)

}

func TeamToString(team pb.Team) string {
	switch team {
	case pb.Team_MAFIA:
		return "mafia"
	case pb.Team_CIVILIANS:
		return "civilians"
	default:
		return "unknown"
	}
}

func RoleToString(role pb.Role) string {
	switch role {
	case pb.Role_SHERIFF:
		return "sheriff"
	case pb.Role_CIVILIAN:
		return "civilian"
	case pb.Role_MAFIA_ROLE:
		return "mafia"
	default:
		return "unknown"
	}
}

func PlayerToString(player *pb.Player) string {
	return fmt.Sprintf("player %s, role: %s, alive: %t", player.Username, RoleToString(player.Role), player.Liveness)
}
