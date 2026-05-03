package middleware

import (
	"net/http"

	"bilim-quiz/internal/auth"
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
