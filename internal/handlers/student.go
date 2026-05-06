package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"bilim-quiz/internal/repository"
)

type StudentHandler struct {
	sessions  *repository.SessionRepo
	questions *repository.QuestionRepo
}

func NewStudentHandler(sessions *repository.SessionRepo, questions *repository.QuestionRepo) *StudentHandler {
	return &StudentHandler{sessions: sessions, questions: questions}
}

type JoinPageData struct {
	Pin string
}

type LobbyStudentData struct {
	Pin string
}

type PlayStudentData struct {
	Pin      string
	PlayerID int
}

type ResultStudentData struct {
	Player       *repository.SessionPlayer
	Rank         int
	TotalPlayers int
	Answers      []repository.PlayerAnswer
}

func (h *StudentHandler) JoinPage(w http.ResponseWriter, r *http.Request) {
	pin := r.URL.Query().Get("pin")
	Render(w, r, "join.html", JoinPageData{Pin: pin})
}

func (h *StudentHandler) CheckPin(w http.ResponseWriter, r *http.Request) {
	pin := r.FormValue("pin")
	if len(pin) != 6 {
		http.Error(w, "invalid pin length", http.StatusBadRequest)
		return
	}
	sess, err := h.sessions.GetByPin(r.Context(), pin)
	if err != nil || sess.Status == "finished" {
		http.Error(w, "Session not found or already finished", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *StudentHandler) LobbyPage(w http.ResponseWriter, r *http.Request) {
	pin := chi.URLParam(r, "pin")
	sess, err := h.sessions.GetByPin(r.Context(), pin)
	if err != nil || sess.Status == "finished" {
		http.Redirect(w, r, "/join?error=1", http.StatusFound)
		return
	}
	Render(w, r, "lobby_student.html", LobbyStudentData{Pin: pin})
}

func (h *StudentHandler) JoinLobby(w http.ResponseWriter, r *http.Request) {
	pin := chi.URLParam(r, "pin")
	nickname := r.FormValue("nickname")
	avatar := r.FormValue("avatar")

	if len(nickname) < 2 {
		http.Error(w, "Name too short", http.StatusBadRequest)
		return
	}

	sess, err := h.sessions.GetByPin(r.Context(), pin)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	if sess.Status != "waiting" {
		http.Error(w, "Game already started", http.StatusConflict)
		return
	}

	if avatar == "" {
		avatar = "🐶"
	}

	player, err := h.sessions.AddPlayer(r.Context(), sess.ID, nickname, avatar)
	if err != nil {
		http.Error(w, "Could not join", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"player_id":  player.ID,
		"session_id": sess.ID,
	})
}

func (h *StudentHandler) PlayPage(w http.ResponseWriter, r *http.Request) {
	pin := chi.URLParam(r, "pin")
	playerID, _ := strconv.Atoi(chi.URLParam(r, "player_id"))
	Render(w, r, "play_student.html", PlayStudentData{Pin: pin, PlayerID: playerID})
}

func (h *StudentHandler) ResultPage(w http.ResponseWriter, r *http.Request) {
	playerID, _ := strconv.Atoi(chi.URLParam(r, "player_id"))

	player, err := h.sessions.GetPlayerByID(r.Context(), playerID)
	if err != nil {
		Render(w, r, "result_student.html", nil)
		return
	}

	answers, _ := h.sessions.GetPlayerAnswers(r.Context(), playerID)
	leaderboard, _ := h.sessions.GetLeaderboard(r.Context(), player.SessionID)

	rank := 1
	for i, p := range leaderboard {
		if p.ID == playerID {
			rank = i + 1
			break
		}
	}

	Render(w, r, "result_student.html", ResultStudentData{
		Player:       player,
		Rank:         rank,
		TotalPlayers: len(leaderboard),
		Answers:      answers,
	})
}

func (h *StudentHandler) RateSession(w http.ResponseWriter, r *http.Request) {
	playerID, _ := strconv.Atoi(chi.URLParam(r, "player_id"))

	var body struct {
		Stars   int    `json:"stars"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Stars < 1 || body.Stars > 5 {
		http.Error(w, "invalid", http.StatusBadRequest)
		return
	}
	runes := []rune(body.Comment)
	if len(runes) > 50 {
		body.Comment = string(runes[:50])
	}

	player, err := h.sessions.GetPlayerByID(r.Context(), playerID)
	if err != nil {
		http.Error(w, "player not found", http.StatusNotFound)
		return
	}

	if err := h.sessions.SaveRating(r.Context(), player.SessionID, playerID, body.Stars, body.Comment); err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
