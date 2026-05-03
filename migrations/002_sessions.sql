-- Game sessions (one per play)
CREATE TABLE IF NOT EXISTS game_sessions (
    id            SERIAL PRIMARY KEY,
    game_id       INTEGER     NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    teacher_id    INTEGER     NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    pin_code      TEXT        NOT NULL UNIQUE,
    status        TEXT        NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting', 'active', 'finished')),
    total_players INTEGER     NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_sessions_pin ON game_sessions(pin_code);
CREATE INDEX IF NOT EXISTS idx_game_sessions_teacher ON game_sessions(teacher_id);

-- Snapshot of questions at the moment the game started (immutable)
CREATE TABLE IF NOT EXISTS session_questions_snapshot (
    id               SERIAL PRIMARY KEY,
    session_id       INTEGER NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    original_id      INTEGER NOT NULL,
    position         INTEGER NOT NULL DEFAULT 0,
    content          TEXT    NOT NULL,
    image_url        TEXT    NOT NULL DEFAULT '',
    youtube_url      TEXT    NOT NULL DEFAULT '',
    youtube_start    INTEGER NOT NULL DEFAULT 0,
    youtube_end      INTEGER NOT NULL DEFAULT 0,
    time_limit       INTEGER NOT NULL DEFAULT 30,
    score_type       TEXT    NOT NULL DEFAULT 'dynamic',
    static_score     INTEGER NOT NULL DEFAULT 1
);

-- Snapshot of answers tied to snapshot questions (immutable)
CREATE TABLE IF NOT EXISTS session_answers_snapshot (
    id                   SERIAL PRIMARY KEY,
    snapshot_question_id INTEGER NOT NULL REFERENCES session_questions_snapshot(id) ON DELETE CASCADE,
    text                 TEXT    NOT NULL,
    is_correct           BOOLEAN NOT NULL DEFAULT FALSE
);

-- Players in a session
CREATE TABLE IF NOT EXISTS session_players (
    id           SERIAL PRIMARY KEY,
    session_id   INTEGER NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    nickname     TEXT    NOT NULL,
    avatar       TEXT    NOT NULL DEFAULT '',
    final_score  INTEGER NOT NULL DEFAULT 0,
    finished_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_session_players_session ON session_players(session_id);

-- Per-question answers from each player (immutable history)
CREATE TABLE IF NOT EXISTS player_answers (
    id                   SERIAL PRIMARY KEY,
    player_id            INTEGER NOT NULL REFERENCES session_players(id) ON DELETE CASCADE,
    snapshot_question_id INTEGER NOT NULL REFERENCES session_questions_snapshot(id) ON DELETE CASCADE,
    selected_answer_text TEXT    NOT NULL DEFAULT '',
    is_correct           BOOLEAN NOT NULL DEFAULT FALSE,
    earned_points        INTEGER NOT NULL DEFAULT 0,
    time_taken_ms        INTEGER NOT NULL DEFAULT 0,
    answered_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
