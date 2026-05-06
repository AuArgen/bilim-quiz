CREATE TABLE IF NOT EXISTS session_ratings (
  id         SERIAL PRIMARY KEY,
  session_id INT NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
  player_id  INT NOT NULL REFERENCES session_players(id) ON DELETE CASCADE,
  stars      SMALLINT NOT NULL CHECK (stars BETWEEN 1 AND 5),
  comment    VARCHAR(50),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (player_id)
);
