package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// ClientRole distinguishes teacher from student connections.
type ClientRole int

const (
	RoleTeacher ClientRole = iota
	RolePlayer
)

// Client wraps a single WebSocket connection.
type Client struct {
	Role     ClientRole
	PlayerID int // only for RolePlayer
	room     *Room
	conn     *websocket.Conn
	send     chan []byte
}

func NewClient(conn *websocket.Conn, role ClientRole, playerID int, room *Room) *Client {
	return &Client{
		Role:     role,
		PlayerID: playerID,
		room:     room,
		conn:     conn,
		send:     make(chan []byte, 256),
	}
}

func (c *Client) SetRoom(r *Room) { c.room = r }

func (c *Client) Send(msg any) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from send on closed channel: %v", r)
		}
	}()
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
		// buffer full — drop
	}
}

// ReadPump pumps messages from WebSocket to room.
func (c *Client) ReadPump() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}

		var msg map[string]any
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		msgType, _ := msg["type"].(string)

		switch msgType {
		case "answer":
			if c.Role == RolePlayer {
				var am AnswerMsg
				if err := json.Unmarshal(raw, &am); err == nil {
					c.room.handleAnswer(c, am)
				}
			}
		case "start_game":
			if c.Role == RoleTeacher {
				c.room.startGame()
			}
		case "kick":
			if c.Role == RoleTeacher {
				pid, _ := msg["player_id"].(float64)
				c.room.kickPlayer(int(pid))
			}
		}
	}
}

// WritePump pumps messages from send channel to WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
