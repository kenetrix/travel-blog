# Single sign-on with OpenID Connect

Travel Blog can log users in through any OpenID Connect provider (Pocket ID,
Authentik, Keycloak, Authelia, ...). It is off until all four required
variables are set. Username/password login keeps working unless you turn it
off.

## Configuration in the app

Administrators can set everything up on the **Administration** page under
*Single sign-on*: issuer, client ID and secret, callback URL, button text,
scopes and whether to turn off password login. **Test** checks that the
provider answers without saving; **Save** does the same check first and then
applies the settings at once, no restart needed. If the provider cannot be
reached, nothing is saved, so a typo cannot lock you out.

The settings are saved in `data/config.db`, a database of its own that also holds other
settings and the user accounts of the installation and is not part of the travel database (`data/travel-blog.db`).
The client secret is stored there in plain text, as SQLite is not encrypted: the file
is readable by its owner only (0600), so keep it out of shared backups. The secret
is never shown again in the app; leave the field empty to keep it. If you prefer not to
keep it on disk, use the environment variables below.

## Configuration by environment

If the four required variables below are set, they win: the page shows them
read-only and the saved configuration is ignored. Use this for configuration
as code or if you do not want a key file on disk.

Set these environment variables for the API (the Proxmox container reads
`/opt/travel-blog/oidc.env`):

| Variable                | Required | Meaning                                                                          |
| ----------------------- | -------- | -------------------------------------------------------------------------------- |
| `OIDC_ISSUER`           | yes      | Issuer URL of the provider; `/.well-known/openid-configuration` must be below it |
| `OIDC_CLIENT_ID`        | yes      | Client ID of the app at the provider                                             |
| `OIDC_CLIENT_SECRET`    | yes      | Client secret                                                                    |
| `OIDC_REDIRECT_URL`     | yes      | `https://<your-host>/api/auth/oidc/callback`, exactly as registered              |
| `OIDC_NAME`             | no       | Text for the button: "Log in with …" (default `SSO`)                             |
| `OIDC_SCOPES`           | no       | Space-separated (default `openid profile email`)                                 |
| `OIDC_DISABLE_PASSWORD` | no       | `true` turns off password login and registration, but only while OIDC works      |

If the provider cannot be reached at start, the app still starts, logs the
problem and retries when the login page asks (at most every 30 seconds).

## Pocket ID

1. In Pocket ID create an **OIDC client** for Travel Blog.
2. Callback URL: `https://<your-host>/api/auth/oidc/callback`.
3. Under allowed user groups choose who may log in (none selected: everyone).
4. Copy client ID and secret:

```bash
OIDC_ISSUER=https://id.example.com
OIDC_CLIENT_ID=...
OIDC_CLIENT_SECRET=...
OIDC_REDIRECT_URL=https://travel.example.com/api/auth/oidc/callback
OIDC_NAME="Pocket ID"
```

Other providers work the same way: register a confidential client with that
callback URL and use the provider's issuer URL.

For local development use `http://localhost:5173/api/auth/oidc/callback`
(Vite proxies `/api`, so the session cookie lands on the right origin).

## How accounts are matched

- A returning user is recognised by issuer + subject, never by email, so
  changing the email at the provider changes nothing.
- A new identity whose email is **verified** by the provider and matches an
  existing account is linked to that account. An unverified match is refused.
- Otherwise an account without a password is created, if registration is
  open. The username comes from `preferred_username` (else the email's first
  part) and gets a suffix like `_2` if taken. A closed registration shows a
  message on the login page.
- Administrators are not created from claims. Promote OIDC users in the admin
  page; the generated `admin` account stays the way in as long as password
  login is on.

Only use a provider you trust to verify emails; for a public provider such as
Google, keep registration closed.
