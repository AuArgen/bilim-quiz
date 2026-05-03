package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"bilim-quiz/internal/ai"
	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/repository"
)

type AIHandler struct {
	teachers  *repository.TeacherRepo
	questions *repository.QuestionRepo
}

func NewAIHandler(t *repository.TeacherRepo, q *repository.QuestionRepo) *AIHandler {
	return &AIHandler{teachers: t, questions: q}
}

func (h *AIHandler) Generate(w http.ResponseWriter, r *http.Request) {
	tid, _ := auth.GetTeacherID(r)

	teacher, err := h.teachers.GetByID(r.Context(), tid)
	if err != nil || teacher.GeminiKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":    "no_key",
			"message":  "Gemini API key not set. Go to dashboard settings to add your key.",
		})
		return
	}

	var req struct {
		Topic  string `json:"topic"`
		GameID int    `json:"game_id"`
		Count  int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Topic == "" || req.GameID == 0 {
		http.Error(w, "topic and game_id required", http.StatusBadRequest)
		return
	}
	if req.Count == 0 {
		req.Count = 5
	}

	generated, err := ai.GenerateQuestions(r.Context(), teacher.GeminiKey, req.Topic, req.Count)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error":   "generation_failed",
			"message": err.Error(),
		})
		return
	}

	// Save to DB
	for _, gq := range generated {
		timeLimit := gq.TimeLimit
		if timeLimit == 0 {
			timeLimit = 30
		}
		q, err := h.questions.Create(r.Context(), &repository.Question{
			GameID:      req.GameID,
			Content:     gq.Content,
			TimeLimit:   timeLimit,
			ScoreType:   "dynamic",
			StaticScore: 1,
		})
		if err != nil {
			continue
		}
		answers := make([]repository.Answer, len(gq.Answers))
		for i, a := range gq.Answers {
			answers[i] = repository.Answer{
				QuestionID: q.ID,
				Text:       a.Text,
				IsCorrect:  a.IsCorrect,
			}
		}
		h.questions.ReplaceAnswers(r.Context(), q.ID, answers)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"generated": len(generated),
		"game_id":   strconv.Itoa(req.GameID),
	})
}
