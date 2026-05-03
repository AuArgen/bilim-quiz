package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepo struct{ db *pgxpool.Pool }

func NewGameRepo(db *pgxpool.Pool) *GameRepo { return &GameRepo{db: db} }

func (r *GameRepo) Create(ctx context.Context, g *Game) (*Game, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO games (teacher_id, title, description)
		 VALUES ($1, $2, $3)
		 RETURNING id, teacher_id, title, description, share_token, created_at, updated_at`,
		g.TeacherID, g.Title, g.Description)
	out := &Game{}
	err := row.Scan(&out.ID, &out.TeacherID, &out.Title, &out.Description,
		&out.ShareToken, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *GameRepo) GetByID(ctx context.Context, id int) (*Game, error) {
	row := r.db.QueryRow(ctx,
		`SELECT g.id, g.teacher_id, g.title, g.description,
		        g.share_token, COUNT(q.id) AS question_count,
		        g.created_at, g.updated_at
		 FROM games g
		 LEFT JOIN questions q ON q.game_id = g.id
		 WHERE g.id = $1
		 GROUP BY g.id`, id)
	out := &Game{}
	err := row.Scan(&out.ID, &out.TeacherID, &out.Title, &out.Description,
		&out.ShareToken, &out.QuestionCount, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *GameRepo) GetByShareToken(ctx context.Context, token string) (*Game, error) {
	row := r.db.QueryRow(ctx,
		`SELECT g.id, g.teacher_id, g.title, g.description,
		        g.share_token, COUNT(q.id) AS question_count,
		        g.created_at, g.updated_at
		 FROM games g
		 LEFT JOIN questions q ON q.game_id = g.id
		 WHERE g.share_token = $1
		 GROUP BY g.id`, token)
	out := &Game{}
	err := row.Scan(&out.ID, &out.TeacherID, &out.Title, &out.Description,
		&out.ShareToken, &out.QuestionCount, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *GameRepo) ListByTeacher(ctx context.Context, teacherID int) ([]Game, error) {
	rows, err := r.db.Query(ctx,
		`SELECT g.id, g.teacher_id, g.title, g.description,
		        g.share_token, COUNT(q.id) AS question_count,
		        g.created_at, g.updated_at
		 FROM games g
		 LEFT JOIN questions q ON q.game_id = g.id
		 WHERE g.teacher_id = $1
		 GROUP BY g.id
		 ORDER BY g.updated_at DESC`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		if err := rows.Scan(&g.ID, &g.TeacherID, &g.Title, &g.Description,
			&g.ShareToken, &g.QuestionCount, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

func (r *GameRepo) Update(ctx context.Context, g *Game) error {
	_, err := r.db.Exec(ctx,
		`UPDATE games SET title=$1, description=$2 WHERE id=$3 AND teacher_id=$4`,
		g.Title, g.Description, g.ID, g.TeacherID)
	return err
}

func (r *GameRepo) Delete(ctx context.Context, id, teacherID int) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM games WHERE id=$1 AND teacher_id=$2`, id, teacherID)
	return err
}
