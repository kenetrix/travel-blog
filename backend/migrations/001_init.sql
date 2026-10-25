-- Every user has their own data. Requests without a session run as the local user 'local'
-- (domain.LocalUserID) in development mode (-local-user); it cannot log in.
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE, -- public name, lowercase
	email TEXT UNIQUE, -- lowercase; NULL for the local user
	password_hash TEXT, -- bcrypt; NULL: no login possible
	first_name TEXT NOT NULL DEFAULT '',
	last_name TEXT NOT NULL DEFAULT '',
	display_name TEXT NOT NULL DEFAULT '',
	bio TEXT NOT NULL DEFAULT '',
	language TEXT NOT NULL DEFAULT 'en', -- BCP 47, e.g. 'de' or 'en-US'
	timezone TEXT NOT NULL DEFAULT 'UTC', -- IANA, e.g. 'Europe/Berlin'
	home_country TEXT NOT NULL DEFAULT '', -- ISO 3166 alpha-3
	home_airport TEXT NOT NULL DEFAULT '', -- IATA
	created_at TEXT NOT NULL -- RFC 3339
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

INSERT INTO users (id, username, display_name, created_at)
VALUES ('local', 'local', 'Local user', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));
INSERT INTO user_settings (user_id) VALUES ('local');

-- Logins; only the SHA-256 hash of the token is stored
CREATE TABLE sessions (
	token_hash TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT NOT NULL, -- RFC 3339
	expires_at TEXT NOT NULL -- RFC 3339
);
CREATE INDEX sessions_user ON sessions (user_id);

CREATE TABLE country_entries (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	iso TEXT NOT NULL,
	status TEXT NOT NULL CHECK (status IN ('visited', 'wishlist')),
	visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
	food TEXT NOT NULL DEFAULT '',
	sightseeing TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	rating INTEGER NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
	tags TEXT NOT NULL DEFAULT '[]', -- JSON array
	PRIMARY KEY (user_id, iso)
);

CREATE TABLE flights (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	from_iata TEXT NOT NULL,
	to_iata TEXT NOT NULL,
	visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
	created_at TEXT NOT NULL, -- RFC 3339
	UNIQUE (user_id, from_iata, to_iata)
);

CREATE TABLE locations (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	country_id TEXT NOT NULL, -- ISO 3166 alpha-3
	name TEXT NOT NULL,
	lat REAL NOT NULL,
	lng REAL NOT NULL,
	visited_at TEXT,
	notes TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public'))
);
CREATE INDEX locations_user_country ON locations (user_id, country_id);

-- Photos belong to a user via their location and inherit its visibility. The file is
-- stored under storage_key ('<user_id>/<uuid>.<ext>').
CREATE TABLE photos (
	id TEXT PRIMARY KEY,
	location_id TEXT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
	storage_key TEXT NOT NULL UNIQUE,
	content_type TEXT NOT NULL,
	caption TEXT NOT NULL DEFAULT '',
	taken_at TEXT,
	created_at TEXT NOT NULL -- RFC 3339
);
CREATE INDEX photos_location ON photos (location_id);
