ALTER TABLE opponents
    ADD COLUMN IF NOT EXISTS identity_key text;

UPDATE opponents
SET identity_key = id::text
WHERE identity_key IS NULL OR btrim(identity_key) = '';

DROP INDEX IF EXISTS opponents_user_lower_name_uq;

ALTER TABLE opponents
    ALTER COLUMN identity_key SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS opponents_user_identity_key_uq
    ON opponents (user_id, identity_key);

CREATE INDEX IF NOT EXISTS opponents_user_lower_name_idx
    ON opponents (user_id, lower(name));
