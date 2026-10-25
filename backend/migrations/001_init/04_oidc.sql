-- External identities (OpenID Connect). An identity is the issuer/subject pair of the provider,
-- never the email address. Accounts that only log in via OIDC have no password (password_hash NULL).
CREATE TABLE oidc_identities (
	issuer TEXT NOT NULL,
	subject TEXT NOT NULL,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT NOT NULL, -- RFC 3339
	PRIMARY KEY (issuer, subject)
);
CREATE INDEX oidc_identities_user ON oidc_identities (user_id);
