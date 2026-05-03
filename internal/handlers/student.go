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

	answers, err := h.sessions.GetPlayerAnswers(r.Context(), playerID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	// Get leaderboard to find rank
	if len(answers) == 0 {
		Render(w, r, "result_student.html", nil)
		return
	}

	sessionID := 0
	if len(answers) > 0 {
		// Get player info
		leaderboard, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)
		rank := 1
		var player *repository.SessionPlayer
		for i, p := range leaderboard {
			if p.ID == playerID {
				rank = i + 1
				pp := p
				player = &pp
				break
			}
		}
		Render(w, r, "result_student.html", ResultStudentData{
			Player:       player,
			Rank:         rank,
			TotalPlayers: len(leaderboard),
			Answers:      answers,
		})
		return
	}

	Render(w, r, "result_student.html", nil)
}
