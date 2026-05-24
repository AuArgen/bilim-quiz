package handlers

import (
	"log"
	"net/http"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/onboarding"
	"bilim-quiz/internal/repository"
)

type AuthHandler struct {
	teachers   *repository.TeacherRepo
	onboarding onboarding.Deps
}

func NewAuthHandler(teachers *repository.TeacherRepo, ob onboarding.Deps) *AuthHandler {
	return &AuthHandler{teachers: teachers, onboarding: ob}
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

	role := "teacher"
	if auth.IsAdminEmail(gUser.Email) {
		role = "admin"
	}

	teacher, isNew, err := h.teachers.Upsert(r.Context(), &repository.Teacher{
		GoogleID:  gUser.ID,
		Email:     gUser.Email,
		Name:      gUser.Name,
		AvatarURL: gUser.AvatarURL,
		Role:      role,
	})
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	if isNew {
		if err := onboarding.SeedDemoGame(r.Context(), h.onboarding, teacher.ID); err != nil {
			log.Printf("onboarding seed failed for teacher %d: %v", teacher.ID, err)
		}
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
