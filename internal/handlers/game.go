package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/repository"
)

type GameHandler struct {
	games     *repository.GameRepo
	questions *repository.QuestionRepo
}

func NewGameHandler(games *repository.GameRepo, questions *repository.QuestionRepo) *GameHandler {
	return &GameHandler{games: games, questions: questions}
}

type GameBuilderData struct {
	Game      *repository.Game
	Questions []repository.Question
}

func (h *GameHandler) NewGame(w http.ResponseWriter, r *http.Request) {
	Render(w, r, "game_builder.html", GameBuilderData{Game: &repository.Game{}})
}

func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	tid, _ := auth.GetTeacherID(r)
	g, err := h.games.Create(r.Context(), &repository.Game{
		TeacherID:   tid,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
	})
	if err != nil {
		http.Error(w, "create error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/games/"+strconv.Itoa(g.ID)+"/edit", http.StatusFound)
}

func (h *GameHandler) EditGame(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	game, err := h.games.GetByID(r.Context(), id)
	if err != nil || game.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	questions, err := h.questions.ListByGame(r.Context(), id)
	if err != nil {
		http.Error(w, "questions error", http.StatusInternalServerError)
		return
	}

	Render(w, r, "game_builder.html", GameBuilderData{Game: game, Questions: questions})
}

func (h *GameHandler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	err := h.games.Update(r.Context(), &repository.Game{
		ID:          id,
		TeacherID:   tid,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
	})
	if err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/games/"+strconv.Itoa(id)+"/edit", http.StatusFound)
}

func (h *GameHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	if err := h.games.Delete(r.Context(), id, tid); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *GameHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	gameID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	tid, _ := auth.GetTeacherID(r)

	game, err := h.games.GetByID(r.Context(), gameID)
	if err != nil || game.TeacherID != tid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	timeLimit, _ := strconv.Atoi(r.FormValue("time_limit"))
	if timeLimit == 0 {
		timeLimit = 30
	}
	staticScore, _ := strconv.Atoi(r.FormValue("static_score"))
	if staticScore == 0 {
		staticScore = 1
	}
	youtubeStart, _ := strconv.Atoi(r.FormValue("youtube_start"))
	youtubeEnd, _ := strconv.Atoi(r.FormValue("youtube_end"))

	scoreType := r.FormValue("score_type")
	if scoreType == "" {
		scoreType = "dynamic"
	}

	answers := parseAnswers(r)
	if err := validateAnswers(answers); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	q, err := h.questions.Create(r.Context(), &repository.Question{
		GameID:       gameID,
		Content:      r.FormValue("content"),
		ImageURL:     r.FormValue("image_url"),
		YoutubeURL:   r.FormValue("youtube_url"),
		YoutubeStart: youtubeStart,
		YoutubeEnd:   youtubeEnd,
		TimeLimit:    timeLimit,
		ScoreType:    scoreType,
		StaticScore:  staticScore,
	})
	if err != nil {
		http.Error(w, "create question error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for i := range answers {
		answers[i].QuestionID = q.ID
	}
	if err := h.questions.ReplaceAnswers(r.Context(), q.ID, answers); err != nil {
		http.Error(w, "answers error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/games/"+strconv.Itoa(gameID)+"/edit?add=1", http.StatusFound)
}

func (h *GameHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	qid, _ := strconv.Atoi(chi.URLParam(r, "qid"))

	timeLimit, _ := strconv.Atoi(r.FormValue("time_limit"))
	if timeLimit == 0 {
		timeLimit = 30
	}
	staticScore, _ := strconv.Atoi(r.FormValue("static_score"))
	if staticScore == 0 {
		staticScore = 1
	}
	youtubeStart, _ := strconv.Atoi(r.FormValue("youtube_start"))
	youtubeEnd, _ := strconv.Atoi(r.FormValue("youtube_end"))
	gameID, _ := strconv.Atoi(r.FormValue("game_id"))

	scoreType := r.FormValue("score_type")
	if scoreType == "" {
		scoreType = "dynamic"
	}

	answers := parseAnswers(r)
	if err := validateAnswers(answers); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.questions.Update(r.Context(), &repository.Question{
		ID:           qid,
		GameID:       gameID,
		Content:      r.FormValue("content"),
		ImageURL:     r.FormValue("image_url"),
		YoutubeURL:   r.FormValue("youtube_url"),
		YoutubeStart: youtubeStart,
		YoutubeEnd:   youtubeEnd,
		TimeLimit:    timeLimit,
		ScoreType:    scoreType,
		StaticScore:  staticScore,
	})
	if err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}

	for i := range answers {
		answers[i].QuestionID = qid
	}
	if err := h.questions.ReplaceAnswers(r.Context(), qid, answers); err != nil {
		http.Error(w, "answers error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/games/"+strconv.Itoa(gameID)+"/edit", http.StatusFound)
}

func (h *GameHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	qid, _ := strconv.Atoi(chi.URLParam(r, "qid"))
	gameID := r.FormValue("game_id")

	if err := h.questions.Delete(r.Context(), qid); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/games/"+gameID+"/edit", http.StatusFound)
}

func parseAnswers(r *http.Request) []repository.Answer {
	texts := r.Form["answer_text[]"]
	corrects := r.Form["answer_correct[]"]

	correctSet := make(map[string]bool)
	for _, c := range corrects {
		correctSet[c] = true
	}

	var answers []repository.Answer
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		answers = append(answers, repository.Answer{
			Text:      text,
			IsCorrect: correctSet[strconv.Itoa(i)],
		})
	}
	return answers
}

func validateAnswers(answers []repository.Answer) error {
	if len(answers) < 2 {
		return errBadQuestion("at least two answers are required")
	}
	for _, answer := range answers {
		if answer.IsCorrect {
			return nil
		}
	}
	return errBadQuestion("correct answer is required")
}

type errBadQuestion string

func (e errBadQuestion) Error() string {
	return string(e)
}
