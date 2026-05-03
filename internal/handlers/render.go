package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/i18n"
	"bilim-quiz/internal/repository"
)

var tmpl *template.Template

func LoadTemplates(dir string) error {
	funcMap := template.FuncMap{
		"t": func(lang, key string) string {
			if i18n.Default == nil {
				return key
			}
			return i18n.Default.T(lang, key)
		},
		"appName": func() string {
			name := os.Getenv("APP_NAME")
			if name == "" {
				return "BilimQuiz"
			}
			return name
		},
		"inc":   func(i int) int { return i + 1 },
		"ms2s": func(ms int) string { return fmt.Sprintf("%.1f", float64(ms)/1000) },
		"jsonPlayers": func(players []repository.SessionPlayer, progress, scores map[int]int) template.JS {
			type pj struct {
				ID       int    `json:"id"`
				Nickname string `json:"nickname"`
				Avatar   string `json:"avatar"`
				Score    int    `json:"score"`
				Answered int    `json:"answered"`
			}
			out := make([]pj, len(players))
			for i, p := range players {
				out[i] = pj{
					ID: p.ID, Nickname: p.Nickname, Avatar: p.Avatar,
					Score:    scores[p.ID],
					Answered: progress[p.ID],
				}
			}
			b, _ := json.Marshal(out)
			return template.JS(b)
		},
		"fmtTime": func(t interface{}) string {
			switch v := t.(type) {
			case time.Time:
				return v.Format("02.01.2006 15:04")
			case *time.Time:
				if v == nil {
					return ""
				}
				return v.Format("02.01.2006 15:04")
			}
			return ""
		},
		"jsonQuestions": func(qs []repository.Question) template.JS {
			type answerJSON struct {
				ID        int    `json:"id"`
				Text      string `json:"text"`
				IsCorrect bool   `json:"is_correct"`
			}
			type questionJSON struct {
				ID           int          `json:"id"`
				Content      string       `json:"content"`
				ImageURL     string       `json:"image_url"`
				YoutubeURL   string       `json:"youtube_url"`
				YoutubeStart int          `json:"youtube_start"`
				YoutubeEnd   int          `json:"youtube_end"`
				TimeLimit    int          `json:"time_limit"`
				ScoreType    string       `json:"score_type"`
				StaticScore  int          `json:"static_score"`
				Answers      []answerJSON `json:"answers"`
			}
			out := make([]questionJSON, len(qs))
			for i, q := range qs {
				ans := make([]answerJSON, len(q.Answers))
				for j, a := range q.Answers {
					ans[j] = answerJSON{ID: a.ID, Text: a.Text, IsCorrect: a.IsCorrect}
				}
				out[i] = questionJSON{
					ID: q.ID, Content: q.Content, ImageURL: q.ImageURL,
					YoutubeURL: q.YoutubeURL, YoutubeStart: q.YoutubeStart,
					YoutubeEnd: q.YoutubeEnd, TimeLimit: q.TimeLimit,
					ScoreType: q.ScoreType, StaticScore: q.StaticScore, Answers: ans,
				}
			}
			b, _ := json.Marshal(out)
			return template.JS(b)
		},
		"jsStr": func(s string) template.JS {
			b, _ := json.Marshal(s)
			return template.JS(b)
		},
	}

	pattern := filepath.Join(dir, "*.html")
	var err error
	tmpl, err = template.New("").Funcs(funcMap).ParseGlob(pattern)
	return err
}

type PageData struct {
	Lang      string
	TeacherID int
	Data      any
}

func Render(w http.ResponseWriter, r *http.Request, name string, data any) {
	lang := i18n.FromContext(r.Context())
	tid, _ := auth.GetTeacherID(r)

	pd := PageData{
		Lang:      lang,
		TeacherID: tid,
		Data:      data,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, pd); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
