package game

import (
	"fmt"
	"math/rand"
	"sync"
)

// Hub manages all active game rooms.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room // pin -> room
}

var Global = &Hub{rooms: make(map[string]*Room)}

func (h *Hub) NewRoom(pin string, teacherConn *Client, sessionID int, questions []SnapshotQuestion) *Room {
	room := &Room{
		Pin:        pin,
		SessionID:  sessionID,
		teacher:    teacherConn,
		players:    make(map[int]*Client),
		questions:  questions,
		state:      StateWaiting,
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
	}
	h.mu.Lock()
	h.rooms[pin] = room
	h.mu.Unlock()
	go room.run()
	return room
}

func (h *Hub) GetRoom(pin string) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[pin]
	return r, ok
}

func (h *Hub) DeleteRoom(pin string) {
	h.mu.Lock()
	delete(h.rooms, pin)
	h.mu.Unlock()
}

func GeneratePin() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
