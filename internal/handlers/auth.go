package handlers

import (
	"net/http"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/repository"
)

type AuthHandler struct {
	teachers *repository.TeacherRepo
}

func NewAuthHandler(teachers *repository.TeacherRepo) *AuthHandler {
	return &AuthHandler{teachers: teachers}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	Render(w, r, "landing.html", nil)
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Support explicit ?next= param (e.g., from shared links)
	if next := r.URL.Query().Get("next"); next != "" {
		auth.SetRedirectAfterLogin(w, r, next)
	}
	url := auth.GetAuthURL(w, r)
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	gUser, err := auth.ExchangeCode(r.Context(), w, r, code, state)
	if err != nil {
		http.Error(w, "OAuth error: "+err.Error(), http.StatusBadRequest)
		return
	}

	teacher, err := h.teachers.Upsert(r.Context(), &repository.Teacher{
		GoogleID:  gUser.ID,
		Email:     gUser.Email,
		Name:      gUser.Name,
		AvatarURL: gUser.AvatarURL,
	})
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	if err := auth.SetTeacherID(w, r, teacher.ID); err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	redirectTo := auth.GetRedirectAfterLogin(r)
	if redirectTo != "" {
		auth.ClearRedirectAfterLogin(w, r)
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSession(w, r)
	http.Redirect(w, r, "/", http.StatusFound)
}
