DROP INDEX IF EXISTS opponents_user_identity_key_uq;
DROP INDEX IF EXISTS opponents_user_lower_name_idx;

CREATE UNIQUE INDEX IF NOT EXISTS opponents_user_lower_name_uq
    ON opponents (user_id, lower(name));

ALTER TABLE opponents
    DROP COLUMN IF EXISTS identity_key;
