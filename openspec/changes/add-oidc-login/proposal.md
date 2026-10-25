# Proposal

## Why

Login has only been possible with a username and password. People who run their own installation usually already have an identity provider (preferably Pocket ID) and want one account for all services instead of yet another password. OIDC as an additional login method enables single sign-on, passkeys (through the provider) and central account management.

## What Changes

- New login method via OpenID Connect (authorization code flow with PKCE), independent of the provider: configuration only needs issuer URL, client ID, client secret and redirect URL; endpoints come from discovery (`/.well-known/openid-configuration`).
- Backend endpoints `GET /api/auth/oidc/config`, `GET /api/auth/oidc/login`, `GET /api/auth/oidc/callback`.
- An external identity (issuer + subject) is linked to a local account; the first login creates an account (if registration is open) or links to an existing account via a verified email.
- After a successful callback everything runs through the existing session (cookie `session`); no provider token is handed to the browser.
- The login page shows "Log in with <provider name>" when OIDC is configured.
- Password login can optionally be turned off (`OIDC_DISABLE_PASSWORD`); by default it stays on.
- The OIDC configuration (issuer, client ID, client secret, redirect URL, display name, scopes, "turn off password login") can also be set in the admin area, with a test button; changes apply without a restart. If the `OIDC_*` variables are set, they win and the form is read-only.
- Without configuration OIDC is completely inactive; existing installations do not change.

## Capabilities

### New Capabilities
- `oidc-login`: login via OpenID Connect against a configurable provider, including account linking and automatic provisioning.

### Modified Capabilities

(none; there are no specs yet and password login stays unchanged)

## Impact

- Backend: new package for the OIDC flow, extensions to `AccountService` and `UserRepository`, migration `003_oidc.sql` (table `oidc_identities`, `password_hash` stays nullable), the router whitelist `/api/auth/` already covers the new routes, configuration via environment in `cmd/api/main.go`.
- API: `api/paths/auth_oidc_config.yaml` plus regenerating `oapi-codegen`; login and callback are browser redirects and only described there.
- Frontend: `AuthPage.vue` and `store.ts`.
- Dependencies: `github.com/coreos/go-oidc/v3`, `golang.org/x/oauth2`.
- Admin configuration: migration `004_oidc_settings.sql` (table `oidc_settings`, client secret encrypted), `GET/PUT/DELETE /api/admin/oidc`, `POST /api/admin/oidc/test`, `AdminPage.vue`, key file `data/secret.key`.
- Docs: `docs/oidc.md` with the environment variables and a Pocket ID example, plus README and Proxmox script.
