package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestionRepo struct{ db *pgxpool.Pool }

func NewQuestionRepo(db *pgxpool.Pool) *QuestionRepo { return &QuestionRepo{db: db} }

func (r *QuestionRepo) ListByGame(ctx context.Context, gameID int) ([]Question, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, game_id, position, content, image_url, youtube_url,
		        youtube_start, youtube_end, time_limit, score_type, static_score, created_at
		 FROM questions WHERE game_id=$1 ORDER BY position`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		if err := rows.Scan(&q.ID, &q.GameID, &q.Position, &q.Content,
			&q.ImageURL, &q.YoutubeURL, &q.YoutubeStart, &q.YoutubeEnd,
			&q.TimeLimit, &q.ScoreType, &q.StaticScore, &q.CreatedAt); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range questions {
		answers, err := r.listAnswers(ctx, questions[i].ID)
		if err != nil {
			return nil, err
		}
		questions[i].Answers = answers
	}
	return questions, nil
}

func (r *QuestionRepo) Create(ctx context.Context, q *Question) (*Question, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO questions
		   (game_id, position, content, image_url, youtube_url,
		    youtube_start, youtube_end, time_limit, score_type, static_score)
		 VALUES ($1,(SELECT COALESCE(MAX(position),0)+1 FROM questions WHERE game_id=$1),
		         $2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, game_id, position, content, image_url, youtube_url,
		           youtube_start, youtube_end, time_limit, score_type, static_score, created_at`,
		q.GameID, q.Content, q.ImageURL, q.YoutubeURL,
		q.YoutubeStart, q.YoutubeEnd, q.TimeLimit, q.ScoreType, q.StaticScore)
	out := &Question{}
	err := row.Scan(&out.ID, &out.GameID, &out.Position, &out.Content,
		&out.ImageURL, &out.YoutubeURL, &out.YoutubeStart, &out.YoutubeEnd,
		&out.TimeLimit, &out.ScoreType, &out.StaticScore, &out.CreatedAt)
	return out, err
}

func (r *QuestionRepo) Update(ctx context.Context, q *Question) error {
	_, err := r.db.Exec(ctx,
		`UPDATE questions SET content=$1, image_url=$2, youtube_url=$3,
		  youtube_start=$4, youtube_end=$5, time_limit=$6,
		  score_type=$7, static_score=$8
		 WHERE id=$9 AND game_id=$10`,
		q.Content, q.ImageURL, q.YoutubeURL, q.YoutubeStart, q.YoutubeEnd,
		q.TimeLimit, q.ScoreType, q.StaticScore, q.ID, q.GameID)
	return err
}

func (r *QuestionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM questions WHERE id=$1`, id)
	return err
}

func (r *QuestionRepo) ReplaceAnswers(ctx context.Context, questionID int, answers []Answer) error {
	_, err := r.db.Exec(ctx, `DELETE FROM answers WHERE question_id=$1`, questionID)
	if err != nil {
		return err
	}
	for _, a := range answers {
		_, err = r.db.Exec(ctx,
			`INSERT INTO answers (question_id, text, is_correct) VALUES ($1,$2,$3)`,
			questionID, a.Text, a.IsCorrect)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *QuestionRepo) listAnswers(ctx context.Context, questionID int) ([]Answer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, question_id, text, is_correct FROM answers WHERE question_id=$1`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var answers []Answer
	for rows.Next() {
		var a Answer
		if err := rows.Scan(&a.ID, &a.QuestionID, &a.Text, &a.IsCorrect); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}
	return answers, rows.Err()
}
