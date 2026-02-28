CREATE TABLE user_stats (
    user_id uuid PRIMARY KEY REFERENCES users(id),
    total_sessions int NOT NULL DEFAULT 0,
    total_matches int NOT NULL DEFAULT 0,
    win_rate numeric NOT NULL DEFAULT 0,
    avg_composure numeric NOT NULL DEFAULT 0,
    avg_rushing_index numeric NOT NULL DEFAULT 0,
    avg_unforced_errors_per_min numeric NOT NULL DEFAULT 0,
    improvement_slope_composure numeric NOT NULL DEFAULT 0,
    improvement_slope_rushing numeric NOT NULL DEFAULT 0,
    last_calculated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE opponent_stats (
    opponent_id uuid PRIMARY KEY REFERENCES opponents(id),
    matches_played int NOT NULL DEFAULT 0,
    win_rate numeric NOT NULL DEFAULT 0,
    avg_composure numeric NOT NULL DEFAULT 0,
    avg_rushing_index numeric NOT NULL DEFAULT 0,
    avg_set_differential numeric NOT NULL DEFAULT 0,
    last_calculated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE weekly_stats (
    user_id uuid NOT NULL REFERENCES users(id),
    week_start_date date NOT NULL,
    avg_composure numeric NOT NULL DEFAULT 0,
    avg_rushing_index numeric NOT NULL DEFAULT 0,
    win_rate numeric NOT NULL DEFAULT 0,
    matches_played int NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, week_start_date)
);

CREATE INDEX weekly_stats_user_week_idx
    ON weekly_stats (user_id, week_start_date DESC);
