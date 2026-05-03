package game

import (
	"sync"
)

// Room manages one active game session.
type Room struct {
	Pin       string
	SessionID int

	mu           sync.RWMutex
	teacher      *Client
	players      map[int]*Client // playerID -> client
	playerStates map[int]*PlayerState
	questions    []SnapshotQuestion
	state        GameState

	broadcast  chan Message
	register   chan *Client
	unregister chan *Client

	// injected save function called when a player answers
	onStart  func(sessionID int)
	onAnswer func(playerID, snapshotQID int, text string, correct bool, points, totalScore, timeTakenMs int)
	onFinish func(sessionID, totalPlayers int)
}

func (r *Room) run() {
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			if client.Role == RolePlayer {
				r.players[client.PlayerID] = client
				if r.playerStates == nil {
					r.playerStates = make(map[int]*PlayerState)
				}
				if _, ok := r.playerStates[client.PlayerID]; !ok {
					r.playerStates[client.PlayerID] = &PlayerState{PlayerID: client.PlayerID, Client: client}
				}
			} else {
				r.teacher = client
			}
			r.mu.Unlock()
			r.broadcastLobbyUpdate()
			if client.Role == RolePlayer {
				r.mu.RLock()
				isPlaying := r.state == StatePlaying
				r.mu.RUnlock()
				if isPlaying {
					r.sendNextQuestionToPlayer(client.PlayerID)
				}
			}

		case client := <-r.unregister:
			r.mu.Lock()
			if client.Role == RolePlayer {
				delete(r.players, client.PlayerID)
			} else if client.Role == RoleTeacher && r.teacher == client {
				r.teacher = nil
			}
			r.mu.Unlock()
			close(client.send)
			r.broadcastLobbyUpdate()

		case msg := <-r.broadcast:
			r.mu.RLock()
			r.sendToTeacher(msg)
			r.mu.RUnlock()
		}
	}
}

func (r *Room) broadcastLobbyUpdate() {
	r.mu.RLock()
	players := make([]map[string]any, 0, len(r.players))
	for id, c := range r.players {
		ps := r.playerStates[id]
		name := ""
		if ps != nil {
			_ = ps
		}
		players = append(players, map[string]any{"player_id": id, "name": name, "client": c != nil})
	}
	count := len(r.players)
	r.mu.RUnlock()

	msg := map[string]any{
		"type":         "lobby_update",
		"player_count": count,
		"players":      players,
	}
	r.broadcastAll(msg)
}

func (r *Room) broadcastAll(msg any) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.players {
		c.Send(msg)
	}
	if r.teacher != nil {
		r.teacher.Send(msg)
	}
}

func (r *Room) broadcastPlayers(msg any) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.players {
		c.Send(msg)
	}
}

func (r *Room) sendToTeacher(msg Message) {
	if r.teacher != nil {
		r.teacher.Send(msg)
	}
}

func (r *Room) sendToPlayer(playerID int, msg any) {
	r.mu.RLock()
	c, ok := r.players[playerID]
	r.mu.RUnlock()
	if ok {
		c.Send(msg)
	}
}

func (r *Room) kickPlayer(playerID int) {
	r.mu.Lock()
	c, ok := r.players[playerID]
	r.mu.Unlock()
	if ok {
		c.Send(map[string]any{"type": "kicked"})
		c.conn.Close()
	}
}

func (r *Room) Register(c *Client) {
	r.register <- c
}

func (r *Room) SetCallbacks(
	onStart func(sessionID int),
	onAnswer func(playerID, sqID int, text string, correct bool, points, totalScore, ms int),
	onFinish func(sessionID, total int),
) {
	r.mu.Lock()
	r.onStart = onStart
	r.onAnswer = onAnswer
	r.onFinish = onFinish
	r.mu.Unlock()
}

func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.players)
}

// LobbyPlayers returns a snapshot of player IDs for the teacher lobby view.
func (r *Room) LobbyPlayers() []int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]int, 0, len(r.players))
	for id := range r.players {
		ids = append(ids, id)
	}
	return ids
}
