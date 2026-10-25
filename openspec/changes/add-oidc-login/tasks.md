# Tasks

## 1. Data model and domain

- [x] 1.1 Migration `backend/migrations/data/001_init/04_oidc.sql`: table `oidc_identities (issuer, subject, user_id, created_at)`, primary key `(issuer, subject)`
- [x] 1.2 `port.UserRepository` and `sqlite/user_repo.go`: `GetByIdentity`, `LinkIdentity`, `GetByEmail`, `Create` with an empty password hash
- [x] 1.3 Repository tests (linking, uniqueness, cascade when the user is deleted), via service tests with a real SQLite database

## 2. Account logic

- [x] 2.1 `AccountService.LoginOIDC(claims)` in `oidc_service.go`: identity -> verified email -> auto-provisioning -> closed (see design.md, decision 5)
- [x] 2.2 Username derivation with normalization, blocklist and a suffix on collision
- [x] 2.3 Service tests for all spec scenarios (known, link, unverified, new, closed, name collision, password login fails for an OIDC account)

## 3. OIDC flow in the backend

- [x] 3.1 Add the dependencies `github.com/coreos/go-oidc/v3` and `golang.org/x/oauth2`
- [x] 3.2 Package `adapter/http/oidc`: configuration from the environment, discovery, signed flow cookie (state, nonce, PKCE, return path), auth URL, code exchange, ID token verification
- [x] 3.3 Accept only relative return paths (open-redirect protection)
- [x] 3.4 Error codes `oidc_failed`, `registration_closed`, `email_unverified` -> redirect `/login?error=...`
- [x] 3.5 Tests with a fake provider (`httptest`): success, wrong state, wrong nonce, expired token, replay, provider error
- [x] 3.6 Wiring in `cmd/api/main.go`; startup without or with a broken configuration stays runnable (log), lazy retry
- [x] 3.7 `OIDC_DISABLE_PASSWORD`: 403 on login/register only with active OIDC; test including "no effect without OIDC"

## 4. API specification

- [x] 4.1 Create `api/paths/auth_oidc_config.yaml` and schema `OidcConfig` (`enabled`, `name`, `passwordLogin`) and include them in `api/openapi.yaml`; login and callback are browser redirects with two cookies and are only described in the contract, their handlers are plain `net/http`
- [x] 4.2 Run `backend/scripts/generate-api.ps1`, implement the handler
- [x] 4.3 Router test: routes reachable without a session, everything else still protected

## 5. Frontend

- [x] 5.1 `store.ts`: `oidcConfig()`
- [x] 5.2 `AuthPage.vue`: button "Log in with {name}" (link to `/api/auth/oidc/login`), hide the password form when `passwordLogin: false`
- [x] 5.3 Translate `?error=` codes into readable messages
- [x] 5.4 Run Prettier, type check

## 6. Docs and operations

- [x] 6.1 README / `backend/README.md`: environment variables, redirect URL, notes on email linking
- [x] 6.2 Pocket ID example (create the client, callback URL, allowed groups); note that other providers work the same way
- [x] 6.3 `proxmox/` script: optional OIDC variables in the service environment
- [ ] 6.4 Manual end-to-end test against a Pocket ID instance (first login, repeat login, logout)

## 7. Configuration in the admin area

- [x] 7.1 Table `oidc_settings` in `001_init/04_oidc.sql`, repository for `oidc_settings` and `secretbox` (AES-GCM, key file `data/secret.key`) with tests
- [x] 7.2 `Provider.Reload` and the configuration source (environment before database); provider and routes always wired, 404 while off; tests
- [x] 7.3 OpenAPI: `GET/PUT/DELETE /admin/oidc`, `POST /admin/oidc/test`; regenerate API and frontend schema
- [x] 7.4 Handler (admin only): validation, discovery before saving, never return the secret, lock-out protection, 409 for the environment source; tests
- [x] 7.5 `AdminPage.vue`: form, test button, note "set by environment variables", secret field only for replacing
- [x] 7.6 Extend `docs/oidc.md` and `backend/README.md` with the admin configuration and `secret.key`
