package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"bilim-quiz/internal/repository"
)

type AdminHandler struct {
	teachers *repository.TeacherRepo
	games    *repository.GameRepo
	sessions *repository.SessionRepo
}

func NewAdminHandler(t *repository.TeacherRepo, g *repository.GameRepo, s *repository.SessionRepo) *AdminHandler {
	return &AdminHandler{teachers: t, games: g, sessions: s}
}

type AdminUsersData struct {
	Stats      repository.AdminStats
	Teachers   []repository.TeacherListItem
	Pagination repository.Pagination
	Query      string
}

type AdminUserData struct {
	Teacher  *repository.Teacher
	Stats    repository.TeacherStats
	Games    []repository.Game
	Sessions []repository.GameSession
	GameMap  map[int]*repository.Game
}

type AdminSessionData struct {
	Session     *repository.GameSession
	Game        *repository.Game
	Teacher     *repository.Teacher
	Players     []repository.SessionPlayer
	Ratings     []repository.SessionRating
	AvgStars    float64
	RatingCount int
}

type AdminPlayerData struct {
	Player    *repository.SessionPlayer
	Session   *repository.GameSession
	Game      *repository.Game
	Teacher   *repository.Teacher
	Questions []repository.SnapshotQuestion
	Answers   []repository.PlayerAnswer
	QMap      map[int]repository.SnapshotQuestion
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	query := r.URL.Query().Get("q")

	stats, err := h.teachers.GetAdminStats(r.Context())
	if err != nil {
		http.Error(w, "admin stats error", http.StatusInternalServerError)
		return
	}

	teachers, pagination, err := h.teachers.ListForAdmin(r.Context(), page, 20, query)
	if err != nil {
		http.Error(w, "admin users error", http.StatusInternalServerError)
		return
	}

	Render(w, r, "admin_users.html", AdminUsersData{
		Stats:      stats,
		Teachers:   teachers,
		Pagination: pagination,
		Query:      query,
	})
}

func (h *AdminHandler) UserDetail(w http.ResponseWriter, r *http.Request) {
	teacherID, _ := strconv.Atoi(chi.URLParam(r, "id"))

	teacher, err := h.teachers.GetByID(r.Context(), teacherID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	stats, err := h.teachers.GetStats(r.Context(), teacherID)
	if err != nil {
		http.Error(w, "stats error", http.StatusInternalServerError)
		return
	}

	games, err := h.games.ListByTeacher(r.Context(), teacherID)
	if err != nil {
		http.Error(w, "games error", http.StatusInternalServerError)
		return
	}

	sessions, err := h.sessions.ListByTeacher(r.Context(), teacherID)
	if err != nil {
		http.Error(w, "sessions error", http.StatusInternalServerError)
		return
	}

	gameMap := map[int]*repository.Game{}
	for i := range games {
		g := games[i]
		gameMap[g.ID] = &g
	}

	Render(w, r, "admin_user.html", AdminUserData{
		Teacher:  teacher,
		Stats:    stats,
		Games:    games,
		Sessions: sessions,
		GameMap:  gameMap,
	})
}

func (h *AdminHandler) SessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "id"))

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	game, _ := h.games.GetByID(r.Context(), sess.GameID)
	teacher, _ := h.teachers.GetByID(r.Context(), sess.TeacherID)
	players, _ := h.sessions.GetLeaderboard(r.Context(), sessionID)
	ratings, _ := h.sessions.GetRatings(r.Context(), sessionID)
	ratingStats, _ := h.sessions.GetRatingStats(r.Context(), sessionID)

	Render(w, r, "admin_session.html", AdminSessionData{
		Session:     sess,
		Game:        game,
		Teacher:     teacher,
		Players:     players,
		Ratings:     ratings,
		AvgStars:    ratingStats.AvgStars,
		RatingCount: ratingStats.Count,
	})
}

func (h *AdminHandler) PlayerDetail(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	playerID, _ := strconv.Atoi(chi.URLParam(r, "player_id"))

	sess, err := h.sessions.GetByID(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	player, err := h.sessions.GetPlayerByID(r.Context(), playerID)
	if err != nil || player.SessionID != sessionID {
		http.Error(w, "player not found", http.StatusNotFound)
		return
	}

	answers, _ := h.sessions.GetPlayerAnswers(r.Context(), playerID)
	questions, _ := h.sessions.GetSnapshotQuestions(r.Context(), sessionID)
	game, _ := h.games.GetByID(r.Context(), sess.GameID)
	teacher, _ := h.teachers.GetByID(r.Context(), sess.TeacherID)

	qMap := map[int]repository.SnapshotQuestion{}
	for _, q := range questions {
		qMap[q.ID] = q
	}

	Render(w, r, "admin_player.html", AdminPlayerData{
		Player:    player,
		Session:   sess,
		Game:      game,
		Teacher:   teacher,
		Questions: questions,
		Answers:   answers,
		QMap:      qMap,
	})
}
