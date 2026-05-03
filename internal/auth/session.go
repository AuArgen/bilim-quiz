package auth

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

const sessionName = "bilimquiz"

func InitStore() {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	store = sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, sessionName)
}

func SetTeacherID(w http.ResponseWriter, r *http.Request, id int) error {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values["teacher_id"] = id
	return sess.Save(r, w)
}

func GetTeacherID(r *http.Request) (int, bool) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return 0, false
	}
	id, ok := sess.Values["teacher_id"].(int)
	return id, ok
}

func ClearSession(w http.ResponseWriter, r *http.Request) error {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}
