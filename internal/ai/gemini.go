package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent"

type GeneratedQuestion struct {
	Content   string   `json:"content"`
	Answers   []Answer `json:"answers"`
	TimeLimit int      `json:"time_limit"`
}

type Answer struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

func GenerateQuestions(ctx context.Context, apiKey, topic string, count int) ([]GeneratedQuestion, error) {
	if count <= 0 {
		count = 5
	}

	prompt := fmt.Sprintf(`Generate %d multiple choice quiz questions about "%s".
Return ONLY valid JSON array. Each item must have:
- "content": question text (string)
- "answers": array of 4 objects with "text" (string) and "is_correct" (bool), exactly one must be true
- "time_limit": seconds (integer, 15-30)

Example:
[{"content":"What is 2+2?","answers":[{"text":"3","is_correct":false},{"text":"4","is_correct":true},{"text":"5","is_correct":false},{"text":"6","is_correct":false}],"time_limit":20}]

Return only the JSON array, no markdown, no explanation.`, count, topic)

	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.7,
			"maxOutputTokens": 2048,
		},
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		geminiURL+"?key="+apiKey, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse gemini response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty gemini response")
	}

	text := result.Candidates[0].Content.Parts[0].Text

	// Strip markdown code fences if present
	if len(text) > 6 && text[:3] == "```" {
		start := 0
		for i, c := range text {
			if c == '\n' {
				start = i + 1
				break
			}
		}
		end := len(text)
		for i := len(text) - 1; i >= 0; i-- {
			if text[i] == '`' {
				end = i
				break
			}
		}
		text = text[start:end]
	}

	var questions []GeneratedQuestion
	if err := json.Unmarshal([]byte(text), &questions); err != nil {
		return nil, fmt.Errorf("parse questions JSON: %w", err)
	}
	return questions, nil
}
