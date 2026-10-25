-- Settings of the installation (key/value), apart from the travel data. Keys of the single
-- sign-on configuration start with 'oidc.'; its client secret is stored in plain text, so
-- the file is created with 0600. OIDC_* environment variables take precedence.
CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
INSERT INTO settings (key, value) VALUES ('registration_open', 'true');
