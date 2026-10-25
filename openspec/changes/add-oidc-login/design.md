# Design

## Context

The backend (Go, hexagonal: `domain` / `application/service` / `adapter`) authenticates with the session cookie `session` (token hash in table `sessions`, 30 days). `AccountService` knows register/login/logout; `users.password_hash` is already nullable (the local user has none). `/api/auth/` is public in the router, and the code generator (`oapi-codegen`) creates the interface from `api/`. Frontend: Vue 3, `AuthPage.vue`, `api/client.ts`. See proposal.md for the motivation.

## Goals / Non-Goals

**Goals:**
- Provider-independent (Pocket ID, Keycloak, Authentik, Authelia, ...) purely via discovery.
- One session mechanism; OIDC only creates a normal session.
- Nothing new is active without configuration.

**Non-Goals:**
- Several providers at once.
- Role/group mapping (admin from a claim) - possible later.
- Single logout / back-channel logout; logout only ends the local session.
- Linking/unlinking identities in the account settings.
- Using the provider's access/refresh tokens (only the ID token and, if needed, UserInfo claims).

## Decisions

1. **Libraries:** `coreos/go-oidc/v3` (discovery, JWKS, ID token verification) and `golang.org/x/oauth2` (code exchange, PKCE via `oauth2.S256ChallengeOption`). Writing our own would be an unnecessary security risk.
2. **Configuration by environment or in the admin area (environment wins):** If the four required variables are set, they alone apply and the form is read-only (configuration as code, no lock-out through the UI). Otherwise the saved configuration applies. Variables (secrets stay out of flags and the process list): `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL` (e.g. `https://host/api/auth/oidc/callback`), optional `OIDC_NAME` (button text, default "SSO"), `OIDC_SCOPES` (default `openid profile email`), `OIDC_DISABLE_PASSWORD`. For Pocket ID the issuer is the base URL of the instance. The client is a confidential client; PKCE is used in addition.
3. **Flow state in a short-lived cookie:** `oidc_flow` (HttpOnly, Secure, SameSite=Lax, Path=`/api/auth/oidc`, 10 min) holds `state`, `nonce`, PKCE verifier and return path, **HMAC-signed** with a key generated randomly at startup. No server state is needed; the callback deletes the cookie, so a replay from the same browser fails. Downside: a restart discards flows in progress (acceptable). SameSite=Lax lets the cookie through on the top-level redirect back from the provider.
4. **Data model** - in `001_init/04_oidc.sql`: `oidc_identities(issuer TEXT, subject TEXT, user_id TEXT REFERENCES users ON DELETE CASCADE, created_at TEXT, PRIMARY KEY (issuer, subject))`. An identity is (issuer, subject), never the email. `users.password_hash` stays NULL for OIDC accounts; `Credentials()` then returns `ok=false`, which uses the existing dummy-hash path without special cases.
5. **Resolving the account** (`AccountService.LoginOIDC(claims)`):
   1. Known identity -> its account.
   2. `email_verified` and the email belongs to an account -> link the identity.
   3. Otherwise, if `registration_open` -> create an account (reusing the `Register` validation; username from `preferred_username`, then the email prefix, then `user`, normalized, with a suffix `_2`, `_3`, ... on collision or blocklist hit; language from `locale` if valid).
   4. Otherwise `ErrForbidden` ("registration is closed").
   An unverified email of an existing account -> `ErrConflict` to prevent account takeover. The error page does not reveal whether an account exists.
6. **HTTP layer:** new package `adapter/http/oidc` (provider, flow, cookie signing). Only `GET /api/auth/oidc/config` is in the OpenAPI contract; `login` and `callback` redirect (302) and set two cookies, which the generated strict handlers cannot express, so they are plain `net/http` routes. Errors in the callback redirect to `/login?error=<code>` (fixed codes `oidc_failed`, `registration_closed`, `email_unverified`), never raw provider text.
7. **Provider initialization at startup;** on failure: log and stay disabled (spec). `Enabled()` retries at most every 30 seconds when something is configured but not initialized, which covers start-order problems (e.g. Docker Compose).
8. **Turning off passwords:** `OIDC_DISABLE_PASSWORD=true` only takes effect while OIDC is initialized; `login` and `register` answer 403 and `config` reports `passwordLogin: false` (the username check stays open). The bootstrap admin (credentials file) remains: the first admin via OIDC is not automatically an admin, which is why password login stays on by default. Document this.
9. **Return path:** only relative paths (starting with `/`, not `//`) are accepted, otherwise `/` - prevents open redirects.
10. **Frontend:** `GET /api/auth/oidc/config` when the login page loads; the button is a plain link to `/api/auth/oidc/login` (full navigation, no fetch). `?error=` is translated into readable text. After the callback the browser lands on `/` and the existing session check takes over.
11. **Saved configuration:** table `oidc_settings (key, value)` (in `001_init/04_oidc.sql`) with keys `issuer`, `client_id`, `client_secret`, `redirect_url`, `name`, `scopes`, `disable_password`. The secret is encrypted with AES-256-GCM (nonce prepended, Base64); the key is in `data/secret.key` (0600, generated randomly on first use). The API never returns the secret (`secretSet: true/false`); an empty field on save keeps it.
12. **Reload without restart:** `Provider` holds its configuration in a swappable form (`Reload(cfg)` under a mutex; discovery is checked before the swap). Handler and router always get the same provider, even if OIDC is not configured at first (`oidc.New` without configuration is a disabled provider); the routes are therefore always registered and answer 404 while OIDC is off. The flow cookie stays valid because the HMAC key is process-wide; flows started before a switch fail at the token check and have to start again.
13. **Admin API:** `GET/PUT/DELETE /api/admin/oidc` (configuration, `source: env|settings|none`, secret only as a flag) and `POST /api/admin/oidc/test` (discovery only, saves nothing). PUT validates required fields and runs discovery first, so nothing is saved if the provider cannot be reached (which also protects "turn off password login"). With `source: env`, PUT and DELETE answer 409.

## Risks / Trade-offs

- **Account takeover via email:** only with `email_verified`; the provider has to be trusted (single-tenant assumption, mention in the docs). Public providers (Google) with open registration are not the target of this self-hosted app.
- **Provider outage** blocks OIDC login; this is why password login stays on by default.
- **Pocket ID:** requires assigning users to the client ("Allowed user groups") - docs hint. Pocket ID delivers the claims `preferred_username`, `email`, `email_verified`; otherwise there are fallbacks.
- **State cookie behind a proxy:** `OIDC_REDIRECT_URL` has to match the externally visible HTTPS URL exactly, otherwise the `state` cookie or the redirect mismatches. Development over `http://localhost` works because `Secure` is allowed there.
- New dependencies grow the binary slightly.
- **Losing `data/secret.key`** makes the saved secret unreadable; enter the secret again. The file belongs in the backup next to the database. Whoever does not want a key on disk uses the environment variables.
- **The admin account as leverage:** an administrator can switch the provider. This is intended; the application trusts administrators with user management anyway.
