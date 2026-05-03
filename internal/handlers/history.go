package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/repository"
)

type HistoryHandler struct {
	sessions  *repository.SessionRepo
	games     *repository.GameRepo
	teachers  *repository.TeacherRepo
}

func NewHistoryHandler(s *repository.SessionRepo, g *repository.GameRepo, t *repository.TeacherRepo) *HistoryHandler {
	return &HistoryHandler{sessions: s, games: g, teachers: t}
}

type HistoryListData struct {
	Sessions []repository.GameSession
	GameMap  map[int]*repository.Game
}

type HistorySessionData struct {
	Session  *repository.GameSession
	Game     *repository.Game
	Players  []repository.SessionPlayer
}

type HistoryPlayerData struct {
	Player    *repository.SessionPlayer
	Session   *repository.GameSession
	Questions []repository.SnapshotQuestion
	Answers   []repository.PlayerAnswer
	QMap      map[int]repository.SnapshotQuestion
}

func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	tid, _ := auth.GetTeacherID(r)
	sessions, err := h.sessions.ListByTeacher(r.Context(), tid)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	gameMap := map[int]*repository.Game{}
	for _, s := range sessions {
		if _, ok := gameMap[s.GameID]; !ok {
			g, err := h.games.GetByID(r.Context(), s.GameID)
			if err == nil {
				gameMap[s.GameID] = g
			}
		}
	}

	Render(w, r, "history_list.html", HistoryListData{Sessions: sessions, GameMap: gameMap})
}

func (h *HistoryHandler) SessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	g, _ := h.games.GetByID(r.Context(), sess.GameID)
	players, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)

	Render(w, r, "history_session.html", HistorySessionData{
		Session: sess, Game: g, Players: players,
	})
}

func (h *HistoryHandler) PlayerDetail(w http.ResponseWriter, r *http.Request) {
	playerID, _ := strconv.Atoi(chi.URLParam(r, "player_id"))
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	answers, _ := h.sessions.GetPlayerAnswers(r.Context(), playerID)
	questions, _ := h.sessions.GetSnapshotQuestions(r.Context(), sessionID)
	players, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)

	qMap := map[int]repository.SnapshotQuestion{}
	for _, q := range questions {
		qMap[q.ID] = q
	}

	var player *repository.SessionPlayer
	for _, p := range players {
		if p.ID == playerID {
			pp := p
			player = &pp
			break
		}
	}

	Render(w, r, "history_player.html", HistoryPlayerData{
		Player:    player,
		Session:   sess,
		Questions: questions,
		Answers:   answers,
		QMap:      qMap,
	})
}
