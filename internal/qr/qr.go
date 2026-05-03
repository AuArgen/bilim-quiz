package qr

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	goqr "github.com/skip2/go-qrcode"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	pin := chi.URLParam(r, "pin")
	content := "http://" + r.Host + "/join?pin=" + pin

	png, err := goqr.Encode(content, goqr.Medium, 200)
	if err != nil {
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(png)
}
