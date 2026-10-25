-- Roles: the first user is created at startup (admin); further
-- administrators are appointed by an administrator.
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin'));
-- 1: the password is generated or set by an administrator and
-- must be changed at the next login
ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0;

-- Settings of the whole installation (key/value)
CREATE TABLE app_settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
INSERT INTO app_settings (key, value) VALUES ('registration_open', 'true');
