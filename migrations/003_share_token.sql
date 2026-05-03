ALTER TABLE games ADD COLUMN IF NOT EXISTS share_token UUID NOT NULL DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX IF NOT EXISTS games_share_token_idx ON games(share_token);
