package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct{ db *pgxpool.Pool }

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(ctx context.Context, gameID, teacherID int, pin string) (*GameSession, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO game_sessions (game_id, teacher_id, pin_code)
		 VALUES ($1,$2,$3)
		 RETURNING id, game_id, teacher_id, pin_code, status, total_players,
		           started_at, finished_at, created_at`,
		gameID, teacherID, pin)
	s := &GameSession{}
	err := row.Scan(&s.ID, &s.GameID, &s.TeacherID, &s.PinCode, &s.Status,
		&s.TotalPlayers, &s.StartedAt, &s.FinishedAt, &s.CreatedAt)
	return s, err
}

func (r *SessionRepo) GetByPin(ctx context.Context, pin string) (*GameSession, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, game_id, teacher_id, pin_code, status, total_players,
		        started_at, finished_at, created_at
		 FROM game_sessions WHERE pin_code=$1`, pin)
	s := &GameSession{}
	err := row.Scan(&s.ID, &s.GameID, &s.TeacherID, &s.PinCode, &s.Status,
		&s.TotalPlayers, &s.StartedAt, &s.FinishedAt, &s.CreatedAt)
	return s, err
}

func (r *SessionRepo) GetByID(ctx context.Context, id int) (*GameSession, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, game_id, teacher_id, pin_code, status, total_players,
		        started_at, finished_at, created_at
		 FROM game_sessions WHERE id=$1`, id)
	s := &GameSession{}
	err := row.Scan(&s.ID, &s.GameID, &s.TeacherID, &s.PinCode, &s.Status,
		&s.TotalPlayers, &s.StartedAt, &s.FinishedAt, &s.CreatedAt)
	return s, err
}

func (r *SessionRepo) ListByTeacher(ctx context.Context, teacherID int) ([]GameSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, game_id, teacher_id, pin_code, status, total_players,
		        started_at, finished_at, created_at
		 FROM game_sessions WHERE teacher_id=$1 ORDER BY created_at DESC`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []GameSession
	for rows.Next() {
		var s GameSession
		if err := rows.Scan(&s.ID, &s.GameID, &s.TeacherID, &s.PinCode, &s.Status,
			&s.TotalPlayers, &s.StartedAt, &s.FinishedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepo) SetStatus(ctx context.Context, id int, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE game_sessions SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *SessionRepo) Start(ctx context.Context, id int) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE game_sessions SET status='active', started_at=$1 WHERE id=$2`, now, id)
	return err
}

func (r *SessionRepo) Finish(ctx context.Context, id, totalPlayers int) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE game_sessions SET status='finished', finished_at=$1, total_players=$2 WHERE id=$3`,
		now, totalPlayers, id)
	return err
}

func (r *SessionRepo) CreateSnapshot(ctx context.Context, sessionID int, questions []Question) ([]SnapshotQuestion, error) {
	var snapshots []SnapshotQuestion
	for _, q := range questions {
		row := r.db.QueryRow(ctx,
			`INSERT INTO session_questions_snapshot
			   (session_id, original_id, position, content, image_url, youtube_url,
			    youtube_start, youtube_end, time_limit, score_type, static_score)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			 RETURNING id, session_id, original_id, position, content, image_url,
			           youtube_url, youtube_start, youtube_end, time_limit, score_type, static_score`,
			sessionID, q.ID, q.Position, q.Content, q.ImageURL, q.YoutubeURL,
			q.YoutubeStart, q.YoutubeEnd, q.TimeLimit, q.ScoreType, q.StaticScore)

		var sq SnapshotQuestion
		if err := row.Scan(&sq.ID, &sq.SessionID, &sq.OriginalID, &sq.Position,
			&sq.Content, &sq.ImageURL, &sq.YoutubeURL, &sq.YoutubeStart, &sq.YoutubeEnd,
			&sq.TimeLimit, &sq.ScoreType, &sq.StaticScore); err != nil {
			return nil, err
		}

		for _, a := range q.Answers {
			var sa SnapshotAnswer
			err := r.db.QueryRow(ctx,
				`INSERT INTO session_answers_snapshot (snapshot_question_id, text, is_correct)
				 VALUES ($1,$2,$3) RETURNING id, snapshot_question_id, text, is_correct`,
				sq.ID, a.Text, a.IsCorrect,
			).Scan(&sa.ID, &sa.SnapshotQuestionID, &sa.Text, &sa.IsCorrect)
			if err != nil {
				return nil, err
			}
			sq.Answers = append(sq.Answers, sa)
		}
		snapshots = append(snapshots, sq)
	}
	return snapshots, nil
}

func (r *SessionRepo) AddPlayer(ctx context.Context, sessionID int, nickname, avatar string) (*SessionPlayer, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO session_players (session_id, nickname, avatar)
		 VALUES ($1,$2,$3)
		 RETURNING id, session_id, nickname, avatar, final_score, finished_at`,
		sessionID, nickname, avatar)
	p := &SessionPlayer{}
	err := row.Scan(&p.ID, &p.SessionID, &p.Nickname, &p.Avatar, &p.FinalScore, &p.FinishedAt)
	return p, err
}

func (r *SessionRepo) UpdatePlayerScore(ctx context.Context, playerID, score int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE session_players SET final_score=$1 WHERE id=$2`, score, playerID)
	return err
}

func (r *SessionRepo) FinishPlayer(ctx context.Context, playerID, score int) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE session_players SET final_score=$1, finished_at=$2 WHERE id=$3`,
		score, now, playerID)
	return err
}

func (r *SessionRepo) SavePlayerAnswer(ctx context.Context, pa *PlayerAnswer) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO player_answers
		   (player_id, snapshot_question_id, selected_answer_text, is_correct, earned_points, time_taken_ms)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		pa.PlayerID, pa.SnapshotQuestionID, pa.SelectedAnswerText,
		pa.IsCorrect, pa.EarnedPoints, pa.TimeTakenMs)
	return err
}

func (r *SessionRepo) GetLeaderboard(ctx context.Context, sessionID int) ([]SessionPlayer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, session_id, nickname, avatar, final_score, finished_at
		 FROM session_players WHERE session_id=$1
		 ORDER BY final_score DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players []SessionPlayer
	for rows.Next() {
		var p SessionPlayer
		if err := rows.Scan(&p.ID, &p.SessionID, &p.Nickname, &p.Avatar,
			&p.FinalScore, &p.FinishedAt); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func (r *SessionRepo) GetPlayerAnswers(ctx context.Context, playerID int) ([]PlayerAnswer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, player_id, snapshot_question_id, selected_answer_text,
		        is_correct, earned_points, time_taken_ms, answered_at
		 FROM player_answers WHERE player_id=$1 ORDER BY answered_at`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var answers []PlayerAnswer
	for rows.Next() {
		var a PlayerAnswer
		if err := rows.Scan(&a.ID, &a.PlayerID, &a.SnapshotQuestionID,
			&a.SelectedAnswerText, &a.IsCorrect, &a.EarnedPoints,
			&a.TimeTakenMs, &a.AnsweredAt); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}
	return answers, rows.Err()
}

func (r *SessionRepo) GetSnapshotQuestions(ctx context.Context, sessionID int) ([]SnapshotQuestion, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, session_id, original_id, position, content, image_url,
		        youtube_url, youtube_start, youtube_end, time_limit, score_type, static_score
		 FROM session_questions_snapshot WHERE session_id=$1 ORDER BY position`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var questions []SnapshotQuestion
	for rows.Next() {
		var q SnapshotQuestion
		if err := rows.Scan(&q.ID, &q.SessionID, &q.OriginalID, &q.Position,
			&q.Content, &q.ImageURL, &q.YoutubeURL, &q.YoutubeStart, &q.YoutubeEnd,
			&q.TimeLimit, &q.ScoreType, &q.StaticScore); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range questions {
		ansRows, err := r.db.Query(ctx,
			`SELECT id, snapshot_question_id, text, is_correct
			 FROM session_answers_snapshot WHERE snapshot_question_id=$1`, questions[i].ID)
		if err != nil {
			return nil, err
		}
		for ansRows.Next() {
			var a SnapshotAnswer
			if err := ansRows.Scan(&a.ID, &a.SnapshotQuestionID, &a.Text, &a.IsCorrect); err != nil {
				ansRows.Close()
				return nil, err
			}
			questions[i].Answers = append(questions[i].Answers, a)
		}
		ansRows.Close()
	}
	return questions, nil
}
