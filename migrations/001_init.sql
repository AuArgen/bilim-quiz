-- Teachers (Google OAuth users)
CREATE TABLE IF NOT EXISTS teachers (
    id          SERIAL PRIMARY KEY,
    google_id   TEXT        NOT NULL UNIQUE,
    email       TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    avatar_url  TEXT        NOT NULL DEFAULT '',
    language    TEXT        NOT NULL DEFAULT 'ky',
    gemini_key  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Games
CREATE TABLE IF NOT EXISTS games (
    id          SERIAL PRIMARY KEY,
    teacher_id  INTEGER     NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Questions
CREATE TABLE IF NOT EXISTS questions (
    id            SERIAL PRIMARY KEY,
    game_id       INTEGER     NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    position      INTEGER     NOT NULL DEFAULT 0,
    content       TEXT        NOT NULL,
    image_url     TEXT        NOT NULL DEFAULT '',
    youtube_url   TEXT        NOT NULL DEFAULT '',
    youtube_start INTEGER     NOT NULL DEFAULT 0,
    youtube_end   INTEGER     NOT NULL DEFAULT 0,
    time_limit    INTEGER     NOT NULL DEFAULT 30,
    score_type    TEXT        NOT NULL DEFAULT 'dynamic' CHECK (score_type IN ('dynamic', 'static')),
    static_score  INTEGER     NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Answers
CREATE TABLE IF NOT EXISTS answers (
    id          SERIAL PRIMARY KEY,
    question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    text        TEXT    NOT NULL,
    is_correct  BOOLEAN NOT NULL DEFAULT FALSE
);

-- Auto-update updated_at on games
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
