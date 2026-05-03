package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/game"
	"bilim-quiz/internal/repository"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSHandler struct {
	sessions  *repository.SessionRepo
	questions *repository.QuestionRepo
}

func NewWSHandler(sessions *repository.SessionRepo, questions *repository.QuestionRepo) *WSHandler {
	return &WSHandler{sessions: sessions, questions: questions}
}

// TeacherLobbyWS — teacher opens a new game room.
func (h *WSHandler) TeacherLobbyWS(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "session_id"))
	tid, _ := auth.GetTeacherID(r)

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Load snapshot questions
	snapQs, err := h.sessions.GetSnapshotQuestions(r.Context(), sessionID)
	if err != nil {
		conn.Close()
		return
	}
	gqs := repoToGameQuestions(snapQs)

	client := game.NewClient(conn, game.RoleTeacher, 0, nil)
	room, ok := game.Global.GetRoom(sess.PinCode)
	if !ok {
		room = game.Global.NewRoom(sess.PinCode, client, sessionID, gqs)
	}
	client.SetRoom(room)

	room.SetCallbacks(
		func(sid int) {
			h.sessions.Start(context.Background(), sid)
		},
		func(playerID, sqID int, text string, correct bool, points, totalScore, ms int) {
			ctx := context.Background()
			h.sessions.SavePlayerAnswer(ctx, &repository.PlayerAnswer{
				PlayerID: playerID, SnapshotQuestionID: sqID,
				SelectedAnswerText: text, IsCorrect: correct,
				EarnedPoints: points, TimeTakenMs: ms,
			})
			h.sessions.UpdatePlayerScore(ctx, playerID, totalScore)
		},
		func(sid, total int) {
			h.sessions.Finish(context.Background(), sid, total)
		},
	)

	room.Register(client)
	go client.WritePump()
	client.ReadPump()
}

// PlayerWS — student connects to an existing room.
func (h *WSHandler) PlayerWS(w http.ResponseWriter, r *http.Request) {
	pin := chi.URLParam(r, "pin")
	playerID, _ := strconv.Atoi(chi.URLParam(r, "player_id"))

	room, ok := game.Global.GetRoom(pin)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := game.NewClient(conn, game.RolePlayer, playerID, room)
	room.Register(client)
	go client.WritePump()
	client.ReadPump()
}

func repoToGameQuestions(qs []repository.SnapshotQuestion) []game.SnapshotQuestion {
	out := make([]game.SnapshotQuestion, len(qs))
	for i, q := range qs {
		ans := make([]game.SnapshotAnswer, len(q.Answers))
		for j, a := range q.Answers {
			ans[j] = game.SnapshotAnswer{ID: a.ID, Text: a.Text, IsCorrect: a.IsCorrect}
		}
		out[i] = game.SnapshotQuestion{
			ID: q.ID, OriginalID: q.OriginalID, Position: q.Position,
			Content: q.Content, ImageURL: q.ImageURL,
			YoutubeURL: q.YoutubeURL, YoutubeStart: q.YoutubeStart, YoutubeEnd: q.YoutubeEnd,
			TimeLimit: q.TimeLimit, ScoreType: q.ScoreType, StaticScore: q.StaticScore,
			Answers: ans,
		}
	}
	return out
}
