package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/game"
	"bilim-quiz/internal/repository"
)

type PlayHandler struct {
	games    *repository.GameRepo
	sessions *repository.SessionRepo
	questions *repository.QuestionRepo
}

func NewPlayHandler(g *repository.GameRepo, s *repository.SessionRepo, q *repository.QuestionRepo) *PlayHandler {
	return &PlayHandler{games: g, sessions: s, questions: q}
}

type LobbyTeacherData struct {
	Session  *repository.GameSession
	Game     *repository.Game
	Players  []repository.SessionPlayer
	QRData   string
}

type MonitorData struct {
	Session   *repository.GameSession
	Questions []repository.SnapshotQuestion
	Progress  map[int]int
	Scores    map[int]int
	Players   []repository.SessionPlayer
}

type PodiumData struct {
	Session     *repository.GameSession
	Players     []repository.SessionPlayer
	Questions   []repository.SnapshotQuestion
	DurationStr string
}

// StartSession — teacher presses Play, creates session + snapshot.
func (h *PlayHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	gameID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	g, err := h.games.GetByID(r.Context(), gameID)
	if err != nil || g.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	questions, err := h.questions.ListByGame(r.Context(), gameID)
	if err != nil || len(questions) == 0 {
		http.Error(w, "no questions", http.StatusBadRequest)
		return
	}

	// Generate unique PIN
	pin := ""
	for {
		pin = game.GeneratePin()
		if _, ok := game.Global.GetRoom(pin); !ok {
			break
		}
	}

	sess, err := h.sessions.Create(r.Context(), gameID, tid, pin)
	if err != nil {
		http.Error(w, "create session error", http.StatusInternalServerError)
		return
	}

	if _, err := h.sessions.CreateSnapshot(r.Context(), sess.ID, questions); err != nil {
		http.Error(w, "snapshot error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/teacher/lobby/"+strconv.Itoa(sess.ID), http.StatusFound)
}

func (h *PlayHandler) LobbyPage(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "session_id"))
	tid, _ := auth.GetTeacherID(r)

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	g, _ := h.games.GetByID(r.Context(), sess.GameID)
	players, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)

	Render(w, r, "lobby_teacher.html", LobbyTeacherData{
		Session: sess,
		Game:    g,
		Players: players,
		QRData:  "http://" + r.Host + "/join?pin=" + sess.PinCode,
	})
}

func (h *PlayHandler) MonitorPage(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "session_id"))
	tid, _ := auth.GetTeacherID(r)

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	snapQs, _ := h.sessions.GetSnapshotQuestions(r.Context(), sessionID)
	players, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)

	progress := map[int]int{}
	scores := map[int]int{}
	if room, ok := game.Global.GetRoom(sess.PinCode); ok {
		progress = room.GetPlayerProgress()
		scores = room.GetPlayerScores()
	}

	Render(w, r, "monitor_teacher.html", MonitorData{
		Session:   sess,
		Questions: snapQs,
		Players:   players,
		Progress:  progress,
		Scores:    scores,
	})
}

func (h *PlayHandler) LobbyPlayers(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "session_id"))
	players, err := h.sessions.GetLeaderboard(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	type playerJSON struct {
		ID       int    `json:"id"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	out := make([]playerJSON, len(players))
	for i, p := range players {
		out[i] = playerJSON{ID: p.ID, Nickname: p.Nickname, Avatar: p.Avatar}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *PlayHandler) PodiumPage(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "session_id"))
	tid, _ := auth.GetTeacherID(r)

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	players, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)
	snapQs, _ := h.sessions.GetSnapshotQuestions(r.Context(), sessionID)

	durationStr := ""
	if sess.StartedAt != nil && sess.FinishedAt != nil {
		d := sess.FinishedAt.Sub(*sess.StartedAt)
		durationStr = fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
	}

	Render(w, r, "podium.html", PodiumData{
		Session:     sess,
		Players:     players,
		Questions:   snapQs,
		DurationStr: durationStr,
	})
}
