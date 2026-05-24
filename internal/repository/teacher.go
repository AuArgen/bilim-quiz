package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TeacherRepo struct{ db *pgxpool.Pool }

func NewTeacherRepo(db *pgxpool.Pool) *TeacherRepo { return &TeacherRepo{db: db} }

func (r *TeacherRepo) Upsert(ctx context.Context, t *Teacher) (*Teacher, bool, error) {
	if t.Role == "" {
		t.Role = "teacher"
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO teachers (google_id, email, name, avatar_url, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (google_id) DO UPDATE
		  SET email=$2, name=$3, avatar_url=$4, role=$5
		RETURNING id, google_id, email, name, avatar_url, language, gemini_key, role, created_at, (xmax = 0)`,
		t.GoogleID, t.Email, t.Name, t.AvatarURL, t.Role,
	)
	out := &Teacher{}
	var isNew bool
	err := row.Scan(&out.ID, &out.GoogleID, &out.Email, &out.Name,
		&out.AvatarURL, &out.Language, &out.GeminiKey, &out.Role, &out.CreatedAt, &isNew)
	return out, isNew, err
}

func (r *TeacherRepo) GetByID(ctx context.Context, id int) (*Teacher, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, google_id, email, name, avatar_url, language, gemini_key, role, created_at
		 FROM teachers WHERE id=$1`, id)
	t := &Teacher{}
	err := row.Scan(&t.ID, &t.GoogleID, &t.Email, &t.Name,
		&t.AvatarURL, &t.Language, &t.GeminiKey, &t.Role, &t.CreatedAt)
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

func (r *TeacherRepo) GetAdminStats(ctx context.Context) (AdminStats, error) {
	var s AdminStats
	err := r.db.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM teachers),
		  (SELECT COUNT(*) FROM games),
		  (SELECT COUNT(*) FROM game_sessions),
		  (SELECT COUNT(*) FROM session_players)`,
	).Scan(&s.TotalTeachers, &s.TotalGames, &s.TotalSessions, &s.TotalPlayers)
	return s, err
}

func (r *TeacherRepo) ListForAdmin(ctx context.Context, page, perPage int, query string) ([]TeacherListItem, Pagination, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	search := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	where := ""
	args := []any{}
	if strings.TrimSpace(query) != "" {
		where = "WHERE LOWER(t.name) LIKE $1 OR LOWER(t.email) LIKE $1"
		args = append(args, search)
	}

	countSQL := `SELECT COUNT(*) FROM teachers t ` + where
	var total int
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, Pagination{}, err
	}

	pagination := NewPagination(page, perPage, total)
	page = pagination.Page
	offset := (page - 1) * perPage
	queryArgs := append(args, perPage, offset)
	limitPos := len(args) + 1
	offsetPos := len(args) + 2

	rows, err := r.db.Query(ctx, `
		SELECT
		  t.id, t.google_id, t.email, t.name, t.avatar_url, t.language, t.gemini_key, t.role, t.created_at,
		  COUNT(DISTINCT g.id) AS total_games,
		  COUNT(DISTINCT gs.id) AS total_sessions,
		  COALESCE(SUM(gs.total_players), 0) AS total_players,
		  MAX(COALESCE(gs.created_at, g.updated_at, t.created_at)) AS last_activity
		FROM teachers t
		LEFT JOIN games g ON g.teacher_id = t.id
		LEFT JOIN game_sessions gs ON gs.game_id = g.id
		`+where+`
		GROUP BY t.id
		ORDER BY t.created_at DESC
		LIMIT $`+strconv.Itoa(limitPos)+` OFFSET $`+strconv.Itoa(offsetPos),
		queryArgs...,
	)
	if err != nil {
		return nil, Pagination{}, err
	}
	defer rows.Close()

	var teachers []TeacherListItem
	for rows.Next() {
		var item TeacherListItem
		if err := rows.Scan(
			&item.ID, &item.GoogleID, &item.Email, &item.Name, &item.AvatarURL, &item.Language,
			&item.GeminiKey, &item.Role, &item.CreatedAt,
			&item.Stats.TotalGames, &item.Stats.TotalSessions, &item.Stats.TotalPlayers,
			&item.LastActivity,
		); err != nil {
			return nil, Pagination{}, err
		}
		teachers = append(teachers, item)
	}
	if err := rows.Err(); err != nil {
		return nil, Pagination{}, err
	}

	return teachers, pagination, nil
}

func NewPagination(page, perPage, total int) Pagination {
	totalPages := 1
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	if page > totalPages {
		page = totalPages
	}
	p := Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	}
	p.HasPrev = page > 1
	p.HasNext = page < totalPages
	return p
}
