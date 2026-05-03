package middleware

import (
	"net/http"

	"bilim-quiz/internal/i18n"
)

func Lang(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := ""

		// 1. Query param: ?lang=ru
		if q := r.URL.Query().Get("lang"); q != "" {
			lang = q
		}

		// 2. Cookie
		if lang == "" {
			if c, err := r.Cookie("lang"); err == nil {
				lang = c.Value
			}
		}

		// 3. Accept-Language header
		if lang == "" {
			lang = i18n.DetectLang(r.Header.Get("Accept-Language"))
		}

		// 4. Default
		if lang == "" {
			lang = "ky"
		}

		// Persist in cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "lang",
			Value:    lang,
			Path:     "/",
			MaxAge:   86400 * 365,
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})

		ctx := i18n.WithLang(r.Context(), lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
