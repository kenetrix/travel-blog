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
