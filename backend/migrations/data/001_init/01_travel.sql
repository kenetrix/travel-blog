-- user_id refers to an account in the accounts database, so no foreign key:
-- UserDataRepo.DeleteAll removes a user's rows.
CREATE TABLE country_entries (
	user_id TEXT NOT NULL,
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
	user_id TEXT NOT NULL,
	from_iata TEXT NOT NULL,
	to_iata TEXT NOT NULL,
	visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
	created_at TEXT NOT NULL, -- RFC 3339
	flight_number TEXT, -- canonical, e.g. 'LH400'
	flown_on TEXT, -- YYYY-MM-DD
	duration_minutes INTEGER CHECK (duration_minutes BETWEEN 1 AND 1440),
	aircraft TEXT,
	UNIQUE (user_id, from_iata, to_iata)
);

-- A trip's countries are derived from its stops; it has no country of its own.
CREATE TABLE trips (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	title TEXT NOT NULL,
	starts_on TEXT, -- YYYY-MM-DD
	ends_on TEXT, -- YYYY-MM-DD
	notes TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
	created_at TEXT NOT NULL, -- RFC 3339
	updated_at TEXT NOT NULL -- RFC 3339
);
CREATE INDEX trips_user ON trips (user_id);

-- Locations are the stops of a trip, in visiting order (position). Their visibility
-- follows the trip's.
CREATE TABLE locations (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
	position INTEGER NOT NULL,
	country_id TEXT NOT NULL, -- ISO 3166 alpha-3
	name TEXT NOT NULL,
	lat REAL NOT NULL,
	lng REAL NOT NULL,
	visited_at TEXT, -- YYYY-MM-DD, arrival
	notes TEXT NOT NULL DEFAULT '',
	rating INTEGER NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
	category TEXT NOT NULL DEFAULT '',
	links TEXT NOT NULL DEFAULT '[]', -- JSON array of {url, label}
	visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
	UNIQUE (trip_id, position)
);
CREATE INDEX locations_user_country ON locations (user_id, country_id);
