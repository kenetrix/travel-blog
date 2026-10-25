-- OIDC configuration from the admin area (key/value). The client secret is stored encrypted
-- under 'client_secret' (see adapter/secretbox). OIDC_* environment variables take precedence.
CREATE TABLE oidc_settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
