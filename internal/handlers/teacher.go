package handlers

import (
	"net/http"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/repository"
)

type TeacherHandler struct {
	teachers *repository.TeacherRepo
	games    *repository.GameRepo
}

func NewTeacherHandler(teachers *repository.TeacherRepo, games *repository.GameRepo) *TeacherHandler {
	return &TeacherHandler{teachers: teachers, games: games}
}

type DashboardData struct {
	Teacher *repository.Teacher
	Stats   repository.TeacherStats
	Games   []repository.Game
}

func (h *TeacherHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tid, _ := auth.GetTeacherID(r)

	teacher, err := h.teachers.GetByID(r.Context(), tid)
	if err != nil {
		http.Error(w, "teacher not found", http.StatusInternalServerError)
		return
	}

	stats, err := h.teachers.GetStats(r.Context(), tid)
	if err != nil {
		http.Error(w, "stats error", http.StatusInternalServerError)
		return
	}

	games, err := h.games.ListByTeacher(r.Context(), tid)
	if err != nil {
		http.Error(w, "games error", http.StatusInternalServerError)
		return
	}

	Render(w, r, "dashboard.html", DashboardData{
		Teacher: teacher,
		Stats:   stats,
		Games:   games,
	})
}

func (h *TeacherHandler) SaveGeminiKey(w http.ResponseWriter, r *http.Request) {
	tid, _ := auth.GetTeacherID(r)
	key := r.FormValue("gemini_key")
	if err := h.teachers.UpdateGeminiKey(r.Context(), tid, key); err != nil {
		http.Error(w, "save error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}
