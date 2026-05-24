package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	Filters    AdminUsersFilters
	PageLinks  []AdminPageLink
	SortLinks  AdminSortLinks
	PrevURL    string
	NextURL    string
	ResetLink  string
}

type AdminUsersFilters struct {
	Query       string
	Sort        string
	Order       string
	MinGames    int
	MinSessions int
	MinPlayers  int
}

type AdminPageLink struct {
	Page      int
	URL       string
	IsCurrent bool
}

type AdminSortLinks struct {
	Games          string
	Sessions       string
	Players        string
	Created        string
	GamesActive    bool
	SessionsActive bool
	PlayersActive  bool
	CreatedActive  bool
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
	filters := adminUsersFiltersFromRequest(r)

	stats, err := h.teachers.GetAdminStats(r.Context())
	if err != nil {
		http.Error(w, "admin stats error", http.StatusInternalServerError)
		return
	}

	teachers, pagination, err := h.teachers.ListForAdmin(r.Context(), repository.AdminTeacherListOptions{
		Page:        page,
		PerPage:     20,
		Query:       filters.Query,
		Sort:        filters.Sort,
		Order:       filters.Order,
		MinGames:    filters.MinGames,
		MinSessions: filters.MinSessions,
		MinPlayers:  filters.MinPlayers,
	})
	if err != nil {
		http.Error(w, "admin users error", http.StatusInternalServerError)
		return
	}

	Render(w, r, "admin_users.html", AdminUsersData{
		Stats:      stats,
		Teachers:   teachers,
		Pagination: pagination,
		Filters:    filters,
		PageLinks:  adminPageLinks(r.URL.Query(), pagination),
		SortLinks:  adminSortLinks(r.URL.Query(), filters),
		PrevURL:    adminURLWith(r.URL.Query(), map[string]string{"page": strconv.Itoa(pagination.PrevPage)}),
		NextURL:    adminURLWith(r.URL.Query(), map[string]string{"page": strconv.Itoa(pagination.NextPage)}),
		ResetLink:  "/admin",
	})
}

func adminUsersFiltersFromRequest(r *http.Request) AdminUsersFilters {
	q := r.URL.Query()
	f := AdminUsersFilters{
		Query: strings.TrimSpace(q.Get("q")),
		Sort:  q.Get("sort"),
		Order: q.Get("order"),
	}
	if f.Sort == "" {
		f.Sort = "created"
	}
	if f.Sort != "created" && f.Sort != "games" && f.Sort != "sessions" && f.Sort != "players" && f.Sort != "name" {
		f.Sort = "created"
	}
	if f.Order != "asc" {
		f.Order = "desc"
	}

	f.MinGames = nonNegativeInt(q.Get("min_games"))
	f.MinSessions = nonNegativeInt(q.Get("min_sessions"))
	f.MinPlayers = nonNegativeInt(q.Get("min_players"))
	return f
}

func nonNegativeInt(value string) int {
	n, _ := strconv.Atoi(value)
	if n < 0 {
		return 0
	}
	return n
}

func adminPageLinks(values url.Values, pagination repository.Pagination) []AdminPageLink {
	if pagination.TotalPages <= 1 {
		return nil
	}

	start := pagination.Page - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > pagination.TotalPages {
		end = pagination.TotalPages
		start = end - 4
		if start < 1 {
			start = 1
		}
	}

	links := make([]AdminPageLink, 0, end-start+1)
	for page := start; page <= end; page++ {
		links = append(links, AdminPageLink{
			Page:      page,
			URL:       adminURLWith(values, map[string]string{"page": strconv.Itoa(page)}),
			IsCurrent: page == pagination.Page,
		})
	}
	return links
}

func adminSortLinks(values url.Values, filters AdminUsersFilters) AdminSortLinks {
	return AdminSortLinks{
		Games:          adminSortURL(values, filters, "games"),
		Sessions:       adminSortURL(values, filters, "sessions"),
		Players:        adminSortURL(values, filters, "players"),
		Created:        adminSortURL(values, filters, "created"),
		GamesActive:    filters.Sort == "games",
		SessionsActive: filters.Sort == "sessions",
		PlayersActive:  filters.Sort == "players",
		CreatedActive:  filters.Sort == "created",
	}
}

func adminSortURL(values url.Values, filters AdminUsersFilters, sort string) string {
	order := "desc"
	if filters.Sort == sort && filters.Order == "desc" {
		order = "asc"
	}
	return adminURLWith(values, map[string]string{
		"page":  "1",
		"sort":  sort,
		"order": order,
	})
}

func adminURLWith(values url.Values, updates map[string]string) string {
	next := url.Values{}
	for key, values := range values {
		for _, value := range values {
			if value != "" {
				next.Add(key, value)
			}
		}
	}
	for key, value := range updates {
		next.Del(key)
		if value != "" {
			next.Set(key, value)
		}
	}
	encoded := next.Encode()
	if encoded == "" {
		return "/admin"
	}
	return "/admin?" + encoded
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
