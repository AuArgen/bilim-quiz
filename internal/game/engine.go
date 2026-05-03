package game

import (
	"math"
	"time"
)

// startGame is called by the teacher client via WebSocket "start_game".
func (r *Room) startGame() {
	r.mu.Lock()
	if r.state != StateWaiting {
		r.mu.Unlock()
		return
	}
	r.state = StatePlaying
	r.mu.Unlock()

	if r.onStart != nil {
		go r.onStart(r.SessionID)
	}

	r.broadcastAll(map[string]any{"type": "game_start"})

	// Send questions one by one to each player individually (async mode)
	go r.runAsync()
}

// runAsync sends all questions to all players simultaneously.
// Each player answers at their own pace; no waiting for others.
func (r *Room) runAsync() {
	r.mu.RLock()
	questions := r.questions
	r.mu.RUnlock()

	// Init progress tracking
	r.mu.Lock()
	if r.playerStates == nil {
		r.playerStates = make(map[int]*PlayerState)
	}
	for id := range r.players {
		if _, ok := r.playerStates[id]; !ok {
			r.playerStates[id] = &PlayerState{PlayerID: id}
		}
	}
	r.mu.Unlock()

	// Send first question to all players
	r.sendNextQuestion(-1)

	// Wait until all players finish or timeout
	total := len(questions)
	deadline := time.Duration(0)
	for _, q := range questions {
		deadline += time.Duration(q.TimeLimit+5) * time.Second
	}
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			r.finishGame()
			return
		case <-ticker.C:
			r.mu.RLock()
			allDone := true
			for _, ps := range r.playerStates {
				if ps.QuestionsAnswered < total {
					allDone = false
					break
				}
			}
			r.mu.RUnlock()
			if allDone && len(r.playerStates) > 0 {
				r.finishGame()
				return
			}
		}
	}
}

// sendNextQuestion sends the next unanswered question to a specific player.
func (r *Room) sendNextQuestionToPlayer(playerID int) {
	r.mu.RLock()
	ps, ok := r.playerStates[playerID]
	questions := r.questions
	r.mu.RUnlock()

	if !ok || ps.QuestionsAnswered >= len(questions) {
		score := 0
		if ps != nil {
			score = ps.TotalScore
		}
		r.sendToPlayer(playerID, map[string]any{
			"type":        "game_over",
			"final_score": score,
		})
		return
	}

	q := questions[ps.QuestionsAnswered]
	// Strip is_correct from answers before sending to student
	answers := make([]map[string]any, len(q.Answers))
	for i, a := range q.Answers {
		answers[i] = map[string]any{"id": a.ID, "text": a.Text}
	}

	r.sendToPlayer(playerID, map[string]any{
		"type":  "question",
		"index": ps.QuestionsAnswered,
		"total": len(questions),
		"question": map[string]any{
			"snapshot_id":   q.ID,
			"content":       q.Content,
			"image_url":     q.ImageURL,
			"youtube_url":   q.YoutubeURL,
			"youtube_start": q.YoutubeStart,
			"youtube_end":   q.YoutubeEnd,
			"time_limit":    q.TimeLimit,
			"score_type":    q.ScoreType,
			"answers":       answers,
		},
	})
}

// sendNextQuestion broadcasts the first question to ALL players (game start).
func (r *Room) sendNextQuestion(skipPlayer int) {
	r.mu.RLock()
	playerIDs := make([]int, 0, len(r.players))
	for id := range r.players {
		playerIDs = append(playerIDs, id)
	}
	r.mu.RUnlock()

	for _, id := range playerIDs {
		if id != skipPlayer {
			r.sendNextQuestionToPlayer(id)
		}
	}
}

// handleAnswer processes a student's answer.
func (r *Room) handleAnswer(c *Client, msg AnswerMsg) {
	r.mu.Lock()
	ps, ok := r.playerStates[c.PlayerID]
	if !ok {
		r.mu.Unlock()
		return
	}

	qIdx := ps.QuestionsAnswered
	questions := r.questions
	if qIdx >= len(questions) {
		r.mu.Unlock()
		return
	}

	q := questions[qIdx]
	if q.ID != msg.QuestionID {
		r.mu.Unlock()
		return
	}

	// Find correct answer
	isCorrect := false
	for _, a := range q.Answers {
		if a.Text == msg.Answer && a.IsCorrect {
			isCorrect = true
			break
		}
	}

	// Calculate points
	earned := 0
	if isCorrect {
		if q.ScoreType == "static" {
			earned = q.StaticScore
		} else {
			// Dynamic: max 1000, decreases linearly with time
			timeLimit := q.TimeLimit
			if timeLimit == 0 {
				timeLimit = 30
			}
			timeSec := float64(msg.TimeTakenMs) / 1000.0
			ratio := 1.0 - (timeSec / float64(timeLimit))
			ratio = math.Max(0.1, ratio)
			earned = int(math.Round(1000 * ratio))
		}
	}

	ps.TotalScore += earned
	ps.QuestionsAnswered++
	r.mu.Unlock()

	// Save to DB (non-blocking)
	if r.onAnswer != nil {
		go r.onAnswer(c.PlayerID, q.ID, msg.Answer, isCorrect, earned, ps.TotalScore, msg.TimeTakenMs)
	}

	// Reveal correct answers to this player
	answers := make([]map[string]any, len(q.Answers))
	for i, a := range q.Answers {
		answers[i] = map[string]any{"id": a.ID, "text": a.Text, "is_correct": a.IsCorrect}
	}

	c.Send(map[string]any{
		"type":          "answer_result",
		"is_correct":    isCorrect,
		"earned_points": earned,
		"total_score":   ps.TotalScore,
		"answers":       answers,
	})

	// Notify teacher of progress
	r.broadcast <- Message{
		Type: "player_progress",
		Payload: map[string]any{
			"player_id":          c.PlayerID,
			"questions_answered": ps.QuestionsAnswered,
			"total_score":        ps.TotalScore,
		},
	}

	// Send next question after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		r.sendNextQuestionToPlayer(c.PlayerID)
	}()
}

func (r *Room) finishGame() {
	r.mu.Lock()
	if r.state == StateFinished {
		r.mu.Unlock()
		return
	}
	r.state = StateFinished
	total := len(r.players)
	r.mu.Unlock()

	// Notify all players
	r.mu.RLock()
	for id, c := range r.players {
		ps := r.playerStates[id]
		score := 0
		if ps != nil {
			score = ps.TotalScore
		}
		c.Send(map[string]any{"type": "game_over", "final_score": score})
	}
	r.mu.RUnlock()

	// Notify teacher
	r.broadcast <- Message{Type: "game_finished", Payload: map[string]any{"total_players": total}}

	if r.onFinish != nil {
		r.onFinish(r.SessionID, total)
	}
}

// GetPlayerScores returns current scores for teacher monitor.
func (r *Room) GetPlayerScores() map[int]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[int]int, len(r.playerStates))
	for id, ps := range r.playerStates {
		out[id] = ps.TotalScore
	}
	return out
}

// GetPlayerProgress returns questions answered count per player.
func (r *Room) GetPlayerProgress() map[int]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[int]int, len(r.playerStates))
	for id, ps := range r.playerStates {
		out[id] = ps.QuestionsAnswered
	}
	return out
}
