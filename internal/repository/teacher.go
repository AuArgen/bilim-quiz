package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TeacherRepo struct{ db *pgxpool.Pool }

func NewTeacherRepo(db *pgxpool.Pool) *TeacherRepo { return &TeacherRepo{db: db} }

func (r *TeacherRepo) Upsert(ctx context.Context, t *Teacher) (*Teacher, bool, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO teachers (google_id, email, name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (google_id) DO UPDATE
		  SET email=$2, name=$3, avatar_url=$4
		RETURNING id, google_id, email, name, avatar_url, language, gemini_key, created_at, (xmax = 0)`,
		t.GoogleID, t.Email, t.Name, t.AvatarURL,
	)
	out := &Teacher{}
	var isNew bool
	err := row.Scan(&out.ID, &out.GoogleID, &out.Email, &out.Name,
		&out.AvatarURL, &out.Language, &out.GeminiKey, &out.CreatedAt, &isNew)
	return out, isNew, err
}

func (r *TeacherRepo) GetByID(ctx context.Context, id int) (*Teacher, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, google_id, email, name, avatar_url, language, gemini_key, created_at
		 FROM teachers WHERE id=$1`, id)
	t := &Teacher{}
	err := row.Scan(&t.ID, &t.GoogleID, &t.Email, &t.Name,
		&t.AvatarURL, &t.Language, &t.GeminiKey, &t.CreatedAt)
	return t, err
}

func (r *TeacherRepo) UpdateLanguage(ctx context.Context, id int, lang string) error {
	_, err := r.db.Exec(ctx, `UPDATE teachers SET language=$1 WHERE id=$2`, lang, id)
	return err
}

func (r *TeacherRepo) UpdateGeminiKey(ctx context.Context, id int, key string) error {
	_, err := r.db.Exec(ctx, `UPDATE teachers SET gemini_key=$1 WHERE id=$2`, key, id)
	return err
}

func (r *TeacherRepo) GetStats(ctx context.Context, teacherID int) (TeacherStats, error) {
	var s TeacherStats
	err := r.db.QueryRow(ctx, `
		SELECT
		  COUNT(DISTINCT g.id),
		  COUNT(DISTINCT gs.id),
		  COALESCE(SUM(gs.total_players), 0)
		FROM games g
		LEFT JOIN game_sessions gs ON gs.game_id = g.id
		WHERE g.teacher_id = $1`, teacherID,
	).Scan(&s.TotalGames, &s.TotalSessions, &s.TotalPlayers)
	return s, err
}
