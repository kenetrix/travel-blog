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
