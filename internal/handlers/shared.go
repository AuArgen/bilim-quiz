package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/game"
	"bilim-quiz/internal/repository"
)

type SharedHandler struct {
	games     *repository.GameRepo
	sessions  *repository.SessionRepo
	questions *repository.QuestionRepo
	teachers  *repository.TeacherRepo
}

func NewSharedHandler(
	g *repository.GameRepo,
	s *repository.SessionRepo,
	q *repository.QuestionRepo,
	t *repository.TeacherRepo,
) *SharedHandler {
	return &SharedHandler{games: g, sessions: s, questions: q, teachers: t}
}

type SharedGameData struct {
	Game    *repository.Game
	Author  *repository.Teacher
	Teacher *repository.Teacher
}

// GamePage — GET /shared/{token}: shows the shared game info page (requires auth).
func (h *SharedHandler) GamePage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	tid, _ := auth.GetTeacherID(r)

	g, err := h.games.GetByShareToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Оюн табылган жок", http.StatusNotFound)
		return
	}

	author, _ := h.teachers.GetByID(r.Context(), g.TeacherID)
	me, _ := h.teachers.GetByID(r.Context(), tid)

	Render(w, r, "shared_game.html", SharedGameData{
		Game:    g,
		Author:  author,
		Teacher: me,
	})
}

// StartSession — POST /shared/{token}/start: creates session for logged-in teacher.
func (h *SharedHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	tid, _ := auth.GetTeacherID(r)

	g, err := h.games.GetByShareToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Оюн табылган жок", http.StatusNotFound)
		return
	}

	questions, err := h.questions.ListByGame(r.Context(), g.ID)
	if err != nil || len(questions) == 0 {
		http.Error(w, "Суроолор жок", http.StatusBadRequest)
		return
	}

	pin := ""
	for {
		pin = game.GeneratePin()
		if _, ok := game.Global.GetRoom(pin); !ok {
			break
		}
	}

	sess, err := h.sessions.Create(r.Context(), g.ID, tid, pin)
	if err != nil {
		http.Error(w, "Сессия түзүү катасы", http.StatusInternalServerError)
		return
	}

	if _, err := h.sessions.CreateSnapshot(r.Context(), sess.ID, questions); err != nil {
		http.Error(w, "Snapshot катасы", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/teacher/lobby/"+strconv.Itoa(sess.ID), http.StatusFound)
}
