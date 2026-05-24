package middleware

import (
	"net/http"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/repository"
)

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.GetTeacherID(r)
		if !ok {
			auth.SetRedirectAfterLogin(w, r, r.RequestURI)
			http.Redirect(w, r, "/auth/google", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(teachers *repository.TeacherRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tid, ok := auth.GetTeacherID(r)
			if !ok {
				auth.SetRedirectAfterLogin(w, r, r.RequestURI)
				http.Redirect(w, r, "/auth/google", http.StatusFound)
				return
			}

			teacher, err := teachers.GetByID(r.Context(), tid)
			if err != nil || !auth.IsAdminEmail(teacher.Email) {
				http.Error(w, "admin access required", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
