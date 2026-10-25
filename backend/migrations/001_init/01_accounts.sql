-- Every user has their own data.
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE, -- public name, lowercase
	email TEXT UNIQUE, -- lowercase
	password_hash TEXT, -- bcrypt; NULL: no login possible
	first_name TEXT NOT NULL DEFAULT '',
	last_name TEXT NOT NULL DEFAULT '',
	display_name TEXT NOT NULL DEFAULT '',
	bio TEXT NOT NULL DEFAULT '',
	language TEXT NOT NULL DEFAULT 'en', -- BCP 47, e.g. 'de' or 'en-US'
	timezone TEXT NOT NULL DEFAULT 'UTC', -- IANA, e.g. 'Europe/Berlin'
	home_country TEXT NOT NULL DEFAULT '', -- ISO 3166 alpha-3
	home_airport TEXT NOT NULL DEFAULT '', -- IATA
	color TEXT NOT NULL DEFAULT '', -- avatar colour as #rrggbb; empty: default
	created_at TEXT NOT NULL, -- RFC 3339
	role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
	-- 1: the password is generated or set by an administrator and must be changed at the next login
	must_change_password INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_settings (
	user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	distance_unit TEXT NOT NULL DEFAULT 'km' CHECK (distance_unit IN ('km', 'mi')),
	theme TEXT NOT NULL DEFAULT 'system' CHECK (theme IN ('system', 'light', 'dark')),
	default_view TEXT NOT NULL DEFAULT 'countries' CHECK (default_view IN ('countries', 'flights')),
	-- Visibility of new entries, flights and locations
	default_visibility TEXT NOT NULL DEFAULT 'private' CHECK (default_visibility IN ('private', 'public')),
	-- public: GET /public/{username} shows profile and public data
	profile_visibility TEXT NOT NULL DEFAULT 'private' CHECK (profile_visibility IN ('private', 'public'))
);

-- Logins; only the SHA-256 hash of the token is stored
CREATE TABLE sessions (
	token_hash TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT NOT NULL, -- RFC 3339
	expires_at TEXT NOT NULL -- RFC 3339
);
CREATE INDEX sessions_user ON sessions (user_id);

-- Settings of the whole installation (key/value)
CREATE TABLE app_settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
INSERT INTO app_settings (key, value) VALUES ('registration_open', 'true');
