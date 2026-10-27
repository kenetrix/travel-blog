# Error codes

Every problem response (`application/problem+json`) carries a stable `code`
next to the English `detail`, plus `params` where the message needs values.
The frontend shows the translation of `errors.<code>` from
`frontend/src/i18n/messages/*.ts` and falls back to `detail` for a code it
does not know. Errors without a code of their own have `code: "unknown"`.

```json
{
    "status": 400,
    "title": "Bad Request",
    "detail": "language \"fr\" must be one of en, de, es, ru",
    "code": "invalid_language",
    "params": { "value": "fr" }
}
```

To add an error, return `newError(kind, "<code>", msg, params...)` in
`backend/internal/application/service` and add `errors.<code>` to all four
message catalogs.

## Requests and sessions

| Code                       | Status | Params | Meaning                                          |
| -------------------------- | ------ | ------ | ------------------------------------------------ |
| `bad_request`              | 400    |        | The request does not match the API contract      |
| `internal`                 | 500    |        | Unexpected server error                          |
| `login_required`           | 401    |        | No session                                       |
| `password_change_required` | 403    |        | The password has to be changed first             |
| `body_too_large`           | 413    |        | Request body over the limit                      |
| `too_many_attempts`        | 429    |        | Login or registration rate limit hit             |
| `password_login_off`       | 403    |        | Password login is disabled, use single sign-on   |
| `unknown`                  | any    |        | No specific code; show `detail`                  |

## Accounts

| Code                    | Status | Params         | Meaning                                       |
| ----------------------- | ------ | -------------- | --------------------------------------------- |
| `registration_closed`   | 403    |                | Registration is closed                        |
| `username_taken`        | 409    |                | Username or email already in use              |
| `username_not_allowed`  | 400    |                | Username is on the blocklist                  |
| `wrong_credentials`     | 401    |                | Wrong username or password                    |
| `wrong_password`        | 403    |                | Wrong current password                        |
| `no_password`           | 403    |                | Single sign-on account without a password     |
| `user_not_found`        | 404    |                | No such user                                  |
| `invalid_username`      | 400    |                | Not 3–30 characters a–z, 0–9 or \_            |
| `invalid_email`         | 400    |                | Not an email address                          |
| `invalid_password`      | 400    |                | Under 10 characters or over 72 bytes          |
| `invalid_language`      | 400    | `value`        | Not one of `en`, `de`, `es`, `ru`             |
| `invalid_timezone`      | 400    | `value`        | Not an IANA time zone                         |
| `invalid_color`         | 400    |                | Not a hex colour like `#3a7d5c`               |
| `invalid_distance_unit` | 400    | `value`        | Not `km` or `mi`                              |
| `invalid_theme`         | 400    | `value`        | Unknown theme                                 |
| `invalid_view`          | 400    | `value`        | Unknown start view                            |
| `text_too_long`         | 400    | `field`, `max` | Text field over its length limit              |

## Administration and single sign-on

| Code                     | Status | Params   | Meaning                                         |
| ------------------------ | ------ | -------- | ----------------------------------------------- |
| `admin_only`             | 403    |          | Administrators only                             |
| `invalid_role`           | 400    |          | Role is not `user` or `admin`                   |
| `last_admin`             | 409    |          | The last administrator cannot be demoted        |
| `own_password`           | 400    |          | Use the profile to change your own password     |
| `own_account`            | 400    |          | Use the profile to delete your own account      |
| `invalid_url`            | 400    | `field`  | Not an http(s) URL                              |
| `client_id_required`     | 400    |          | OIDC client ID missing                          |
| `client_secret_required` | 400    |          | OIDC client secret missing                      |
| `oidc_from_env`          | 409    |          | OIDC is set by environment variables            |
| `provider_unreachable`   | 400    | `reason` | The OIDC provider cannot be reached             |
| `invalid_identity`       | 400    |          | Provider identity without issuer or subject     |
| `no_free_username`       | 409    |          | No free username for a new single sign-on user  |
| `identity_linked`        | 409    |          | The identity is linked to another account       |

Failed single sign-on logins redirect to `/login?error=<code>` instead, with
`oidc_failed`, `registration_closed` or `email_unverified`; the frontend
translates those under `oidcErrors.*`.

## Travel data

| Code                    | Status | Params          | Meaning                                    |
| ----------------------- | ------ | --------------- | ------------------------------------------ |
| `invalid_code`          | 400    | `kind`, `value` | Country or airport code is not 3 letters   |
| `invalid_visibility`    | 400    | `value`         | Not `private` or `public`                  |
| `country_not_found`     | 404    |                 | No entry for this country                  |
| `invalid_rating`        | 400    |                 | Rating outside 0–5                         |
| `invalid_status`        | 400    | `value`         | Unknown country status                     |
| `too_many_countries`    | 400    |                 | A flight touches at most two countries     |
| `same_airport`          | 422    |                 | Departure and arrival are the same         |
| `flight_exists`         | 409    | `from`, `to`    | The route was already added                |
| `flight_not_found`      | 404    |                 | No such flight                             |
| `nothing_to_change`     | 400    |                 | Empty flight update                        |
| `invalid_flight_number` | 400    | `value`         | Not an airline code and up to four digits  |
| `invalid_date`          | 400    | `value`         | Not a day between 1900 and 2100            |
| `invalid_duration`      | 400    | `max`           | Duration outside 1 to `max` minutes        |
| `name_required`         | 400    |                 | Location without a name                    |
| `location_not_found`    | 404    |                 | No such location                           |
| `unsupported_image`     | 400    | `value`         | Image type not supported                   |
| `photo_too_large`       | 400    | `max`           | Photo over `max` MB                        |
| `photo_not_found`       | 404    |                 | No such photo                              |
