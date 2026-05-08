package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func UploadPlayerImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Accept base64 data URI from Canvas API
	dataURI := r.FormValue("image_data")
	if dataURI != "" {
		url, err := saveBase64Image(dataURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"url":"%s"}`, url)
		return
	}

	http.Error(w, "no image data", http.StatusBadRequest)
}

func saveBase64Image(dataURI string) (string, error) {
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid data URI")
	}

	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	dir := os.Getenv("PLAYER_IMAGES_DIR")
	if dir == "" {
		dir = "./player_images"
	}

	filename := fmt.Sprintf("%d.jpg", time.Now().UnixNano())
	path := dir + "/" + filename

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/player_images/" + filename, nil
}
