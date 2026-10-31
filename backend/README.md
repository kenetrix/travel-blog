# Backend Setup

Install dependencies:

    go mod download

Start the server (`-dev` is for development only: requests without a session run as the
passwordless administrator `admin`, so you can start without logging in):

    go run ./cmd/api -dev

It listens on `localhost:8080`; `LISTEN_ADDR` (e.g. `:8080`) changes that, as the Docker
image does so nginx in another container can reach it.

All routes except `/api/auth/*` and `/api/public/*` require a login. On first start the
server creates the administrator `admin` with a random password, written to
`data/initial-admin.txt` until it is changed (see below).

## Architecture (hexagonal)

- internal/domain              -> pure business models, no dependencies
- internal/application/port    -> interfaces (ports) the domain defines towards the outside
- internal/application/service -> use cases, using only ports
- internal/adapter/http        -> driving adapter (REST API)
- internal/adapter/repository  -> driven adapter (SQLite implementation of the ports)
- internal/adapter/storage     -> driven adapter (file storage, replaceable)

Swapping MapLibre for another map solution only affects the frontend,
swapping SQLite for Postgres only affects internal/adapter/repository.

## Database migrations

Migrations in `migrations/data/` only go forward and are tracked by SQLite's `user_version`.
Each top-level `.sql` file or folder is one migration, in name order. The folder `001_init/` is
schema version 1, split by area (travel data, photos). Until the `v1.0.0` tag
exists, change it in place and recreate the test database; from `v1.0.0` on,
add `002_<name>.sql`, `003_...` as usual and never edit older ones.

The user accounts (users, their settings, sessions and OIDC identities) live in a second
database, `data/accounts.db`, and the settings of the installation (registration, single
sign-on) in a third, `data/config.db`; their migrations are in `migrations/accounts/` and
`migrations/config/` under the same rules. Deleting `data/travel-blog.db` therefore keeps all
accounts. The databases cannot reference each other, so
`user_id` in the travel data has no foreign key and `UserDataRepo.DeleteAll` removes a user's
rows when the account is deleted.
Before pending migrations run on an existing database, a consistent copy is saved next to
it as `<db>.bak-v<version>`; to roll back, stop the app and put that file back.

## Users and accounts

All travel data belongs to exactly one user (`user_id` in every table, `domain.UserID`).
Repositories and services get the user explicitly on every call; handlers read it
from the request context (`internal/adapter/http/auth`).

- Accounts: username and password (bcrypt); the email is optional. Login sets a
  session cookie (`session`, HttpOnly, Secure, SameSite=Lax, 30 days); the database
  only holds the SHA-256 hash of the token.
- Profile (name, language, time zone, home country, home airport, ...) and settings
  (unit, theme, start view, visibility) come with `GET /api/user`; countries and
  flights come with `GET /api/countries` and `GET /api/flights`, each with its own ETag;
  trips likewise with `GET /api/trips`, a single trip with its stops via `GET /api/trips/{id}`.
- Visibility: entries, flights and trips are `private` or `public`; the stops of a trip
  (`locations`) take the trip's, and photos inherit the one of their stop. With a public
  profile, `GET /api/public/{username}` shows the public entries and flights.
- Photos live in `data/photos/<user>/<uuid>.<ext>`; access checks ownership via the
  stop. Deleting photos, stops, trips and accounts deletes their files too.

## OpenID Connect login

Optional and off without the `OIDC_*` environment variables (setup:
[`docs/oidc.md`](../docs/oidc.md)). `adapter/http/oidc` runs the authorization code flow
with PKCE (`GET /api/auth/oidc/login` and `/callback` are plain `net/http` because they
redirect and set two cookies); `GET /api/auth/oidc/config` is part of the OpenAPI contract.
The result is a normal session. An identity is (issuer, subject) in `oidc_identities`;
`AccountService.LoginOIDC` links it via a verified email or creates an account without
password (only if registration is open). Such accounts still appear in the user list.

The configuration comes from the `OIDC_*` variables (they win) or from the admin area
(`/api/admin/oidc`, configuration database `data/config.db`, not the travel database). `service.OIDCConfigService` checks the
provider via discovery before saving and reloads `oidc.Provider` without a restart. The
client secret is stored in plain text in `config.db` (0600, SQLite is not encrypted) and
is never returned.

## Roles

Every user is `user` or `admin` (`users.role`). At startup `AccountService.EnsureAdmin`
creates the first administrator if there is none: user `admin` with a random password.
The credentials are in `data/initial-admin.txt` (0600) until the password is changed;
`must_change_password` requires that at the first login, and until then the account only
reaches `GET /api/user`, `PUT /api/user/password` and `/api/auth/*` (see
`router.requirePasswordChange`).

Administrators manage users via `/api/admin/*` (role, password reset, delete) and turn
self-registration on or off (`registration_open` in `data/config.db`). The last administrator
stays an administrator.
