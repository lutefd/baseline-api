CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE TABLE opponents (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    name text NOT NULL,
    dominant_hand text NULL CHECK (dominant_hand IN ('left', 'right', 'unknown')),
    play_style text NULL,
    notes text NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE UNIQUE INDEX opponents_user_lower_name_uq
    ON opponents (user_id, lower(name));
CREATE INDEX opponents_user_updated_idx
    ON opponents (user_id, updated_at);
CREATE INDEX opponents_active_idx
    ON opponents (user_id, id)
    WHERE deleted_at IS NULL;

CREATE TABLE sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    opponent_id uuid NULL REFERENCES opponents(id),
    session_name text NOT NULL,
    session_type text NOT NULL CHECK (session_type IN ('class', 'friendly', 'match')),
    date timestamptz NOT NULL,
    duration_minutes int NOT NULL CHECK (duration_minutes > 0),
    rushed_shots int NOT NULL DEFAULT 0,
    unforced_errors int NOT NULL DEFAULT 0,
    long_rallies int NOT NULL DEFAULT 0,
    direction_changes int NOT NULL DEFAULT 0,
    composure int NOT NULL CHECK (composure BETWEEN 1 AND 10),
    focus_text text NULL,
    followed_focus text NULL CHECK (followed_focus IN ('yes', 'partial', 'no')),
    is_match_win boolean NULL,
    notes text NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE INDEX sessions_user_updated_idx
    ON sessions (user_id, updated_at);
CREATE INDEX sessions_user_date_desc_idx
    ON sessions (user_id, date DESC);
CREATE INDEX sessions_active_idx
    ON sessions (user_id, date DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE match_sets (
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    set_number int NOT NULL CHECK (set_number BETWEEN 1 AND 5),
    player_games int NOT NULL CHECK (player_games BETWEEN 0 AND 30),
    opponent_games int NOT NULL CHECK (opponent_games BETWEEN 0 AND 30),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    UNIQUE (session_id, set_number)
);

CREATE INDEX match_sets_session_updated_idx
    ON match_sets (session_id, updated_at);
CREATE INDEX match_sets_active_idx
    ON match_sets (session_id, set_number)
    WHERE deleted_at IS NULL;
