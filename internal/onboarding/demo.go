package onboarding

import (
	"context"

	"bilim-quiz/internal/repository"
)

type Deps struct {
	Games     *repository.GameRepo
	Questions *repository.QuestionRepo
}

// SeedDemoGame creates a sample math game for a newly registered teacher.
func SeedDemoGame(ctx context.Context, deps Deps, teacherID int) error {
	game, err := deps.Games.Create(ctx, &repository.Game{
		TeacherID:   teacherID,
		Title:       "Математика — 8-класс (Демо)",
		Description: "Бул демо оюн. Редактор менен өзүңүз суроо кошо аласыз.",
	})
	if err != nil {
		return err
	}

	type qa struct {
		question string
		answers  [4]string
		correct  int // 0-based index
	}

	questions := []qa{
		{
			"2x + 5 = 11 теңдемесиндеги x нечеге барабар?",
			[4]string{"x = 2", "x = 3", "x = 4", "x = 6"},
			1,
		},
		{
			"Пифагор теоремасы боюнча a = 3, b = 4 болсо, c = ?",
			[4]string{"c = 6", "c = 7", "c = 5", "c = 8"},
			2,
		},
		{
			"√144 нечеге барабар?",
			[4]string{"11", "13", "14", "12"},
			3,
		},
		{
			"Эки санды кошконго 56 болду. Бири 29 болсо, экинчиси?",
			[4]string{"25", "27", "28", "29"},
			1,
		},
		{
			"a² − b² = ?",
			[4]string{"(a − b)²", "(a + b)(a − b)", "(a − b)(a − b)", "(a + b)²"},
			1,
		},
		{
			"x² = 49 болсо, x нечеге барабар?",
			[4]string{"x = 7 гана", "x = ±7", "x = ±49", "x = −7 гана"},
			1,
		},
		{
			"Квадраттын периметри 24 см. Анын аянты?",
			[4]string{"48 см²", "36 см²", "72 см²", "32 см²"},
			1,
		},
		{
			"200дүн 25%и нечеге барабар?",
			[4]string{"40", "45", "50", "55"},
			2,
		},
		{
			"3² + 4² + 5² = ?",
			[4]string{"25", "36", "50", "60"},
			2,
		},
		{
			"Теңдеме системасы: x + y = 10, x − y = 4. x = ?",
			[4]string{"5", "6", "7", "8"},
			2,
		},
	}

	for _, q := range questions {
		created, err := deps.Questions.Create(ctx, &repository.Question{
			GameID:    game.ID,
			Content:   q.question,
			TimeLimit: 30,
			ScoreType: "dynamic",
		})
		if err != nil {
			return err
		}

		var answers []repository.Answer
		for i, text := range q.answers {
			answers = append(answers, repository.Answer{
				Text:      text,
				IsCorrect: i == q.correct,
			})
		}
		if err := deps.Questions.ReplaceAnswers(ctx, created.ID, answers); err != nil {
			return err
		}
	}

	return nil
}
