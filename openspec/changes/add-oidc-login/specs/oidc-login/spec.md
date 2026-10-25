# Spec Delta

## Purpose
Users can log in through any OpenID Connect provider in addition to username and password. The feature is inactive without configuration.

## ADDED Requirements

### Requirement: OIDC is optional and enabled by configuration
The system SHALL offer OIDC login only if issuer URL, client ID, client secret and redirect URL are configured. Without a complete configuration the system behaves as before.

#### Scenario: Not configured
- **WHEN** no OIDC configuration is set
- **THEN** `GET /api/auth/oidc/config` reports `enabled: false` and `GET /api/auth/oidc/login` answers 404

#### Scenario: Configured
- **WHEN** the OIDC configuration is complete and the provider is reachable
- **THEN** `GET /api/auth/oidc/config` reports `enabled: true` and the display name of the provider

#### Scenario: Provider not reachable at startup
- **WHEN** the discovery of the provider fails
- **THEN** the system still starts, OIDC stays disabled, and the error is logged

### Requirement: Login with the authorization code flow and PKCE
The system SHALL perform login with the authorization code flow using PKCE, `state` and `nonce`, and SHALL verify the ID token (signature, issuer, audience, expiry, nonce).

#### Scenario: Start login
- **WHEN** a visitor calls `GET /api/auth/oidc/login`
- **THEN** they are redirected to the authorization endpoint of the provider, and `state`, `nonce` and the PKCE verifier are bound to the browser

#### Scenario: Successful callback
- **WHEN** the provider redirects back to the callback with a valid `code` and a matching `state` and the ID token is valid
- **THEN** a session is created, the session cookie is set and the browser is redirected to the application

#### Scenario: Invalid state or token
- **WHEN** `state` is missing or does not match the browser, the token is invalid, or the provider reports an error
- **THEN** no session is created and the visitor sees an error message on the login page

#### Scenario: Replayed callback
- **WHEN** a callback with an already used `state` is called again
- **THEN** it is rejected

### Requirement: An external identity belongs to exactly one account
The system SHALL map an identity, the pair of issuer and subject, to one local account and SHALL use the same account on every later login.

#### Scenario: Returning user
- **WHEN** a user with a known identity logs in
- **THEN** they are logged in as the same account, even if their email or name changed at the provider

#### Scenario: First login with open registration
- **WHEN** an unknown identity logs in and registration is open
- **THEN** an account without password is created, with the username from `preferred_username` (otherwise the email prefix) and email, first/last name and display name from the claims, and the user is logged in

#### Scenario: First login with closed registration
- **WHEN** an unknown identity logs in and registration is closed
- **THEN** no account is created and the user gets the message that registration is closed

#### Scenario: Username already taken
- **WHEN** the derived username already exists, is blocked or is invalid
- **THEN** the system picks a free, valid name (e.g. with a numeric suffix)

### Requirement: Linking to an existing account only through a verified email
The system SHALL link a new identity to an existing account with the same email only if the provider reports the email as verified (`email_verified: true`).

#### Scenario: Verified email matches
- **WHEN** an unknown identity with the verified email of an existing account logs in
- **THEN** the identity is assigned to that account and the user is logged in as that account

#### Scenario: Unverified email matches
- **WHEN** the email belongs to an existing account but is not reported as verified
- **THEN** the login is rejected and neither linked nor is a new account created

### Requirement: Accounts without password
The system SHALL NOT allow password login for accounts created via OIDC and SHALL NOT require a password change for them.

#### Scenario: Password login for an OIDC account
- **WHEN** someone tries to log in with a password using the username of an account without password
- **THEN** the login fails with the usual error message

#### Scenario: No forced password change
- **WHEN** an OIDC account logs in for the first time
- **THEN** no password change is required

### Requirement: Password login can be turned off
The system SHALL be able to turn off password login and password registration while OIDC is active.

#### Scenario: Password turned off
- **WHEN** password login is turned off and OIDC is active
- **THEN** the login and registration endpoints reject with 403 and the login page shows only the OIDC button

#### Scenario: Turning off has no effect without OIDC
- **WHEN** password login is to be turned off but OIDC is not active
- **THEN** password login stays active so nobody is locked out

### Requirement: The login page offers OIDC
When OIDC is active, the login page SHALL show a button "Log in with <provider name>" that calls the login endpoint.

#### Scenario: Button visible
- **WHEN** `enabled` is true
- **THEN** the login page shows the button; with `enabled: false` it does not

#### Scenario: Error after returning
- **WHEN** the callback redirects to the login page with an error
- **THEN** a readable error message is shown

### Requirement: Source of the configuration
The system SHALL take the OIDC configuration from the environment variables or, if these are not completely set, from the configuration saved in the admin area. If the required variables are set, they alone apply.

#### Scenario: Environment variables set
- **WHEN** all required variables are set
- **THEN** the admin configuration shows `source: env` with the values (without secret) read-only, and saving is rejected with 409

#### Scenario: Only saved configuration
- **WHEN** the required variables are not set and a configuration is saved
- **THEN** the saved configuration applies (`source: settings`)

### Requirement: Configuration in the admin area
The system SHALL allow administrators to read, test and save the OIDC configuration; other users get 403.

#### Scenario: Saving a working configuration
- **WHEN** an administrator saves a configuration whose discovery succeeds
- **THEN** it is saved and applies immediately without a restart; `GET /api/auth/oidc/config` reports `enabled: true`

#### Scenario: Saving a broken configuration
- **WHEN** the discovery fails or required fields are missing
- **THEN** it is rejected with 400 and an error message, and the running configuration stays unchanged

#### Scenario: Test
- **WHEN** an administrator tests the configuration
- **THEN** the discovery of the provider runs and the result is reported without saving

#### Scenario: Removal
- **WHEN** an administrator deletes the saved configuration
- **THEN** OIDC is disabled and password login is active again

### Requirement: The client secret stays secret
The system SHALL never return the client secret through the API and SHALL store it encrypted.

#### Scenario: Reading
- **WHEN** the configuration is read
- **THEN** the response only says whether a secret is set

#### Scenario: Saving without a new secret
- **WHEN** the secret field is empty on save
- **THEN** the existing secret is kept

#### Scenario: Storage
- **WHEN** the secret is saved
- **THEN** it is not in the database in plain text

### Requirement: Lock-out protection when turning off passwords
The system SHALL save "turn off password login" only if the discovery of the configuration has just succeeded.

#### Scenario: Provider not reachable
- **WHEN** "turn off password login" is to be saved but the discovery fails
- **THEN** it is rejected with 400
