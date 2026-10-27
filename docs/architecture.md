# Architecture

Diagrams for [the README](../README.md).

- [How it fits together](#how-it-fits-together)
- [Backend layers](#backend-layers)
- [Authentication](#authentication)

## How it fits together

```mermaid
flowchart LR
    subgraph Client["🖥️ Frontend · Vue 3 + Vite"]
        UI["Panels & pages"] --> Store[("Pinia store")]
        Store --> Map["MapLibre map"]
        Ref["Countries · airports · facts<br/>(reference data)"] --> Store
    end

    subgraph Server["⚙️ Backend · Go"]
        HTTP["HTTP adapter<br/>(generated from OpenAPI)"] --> Svc["Application services"]
        Svc --> Domain["Domain"]
        Svc --> DB[("SQLite")]
        Svc --> Files[("Photo storage")]
    end

    Store -- "GET /api/user · countries · flights<br/>(ETag), small writes" --> HTTP
    Contract{{"📜 api/openapi.yaml"}} -. generates .-> HTTP
    Contract -. generates .-> Store
```

## Backend layers

```mermaid
flowchart TB
    Browser["🌐 Browser"] -- "/api/*" --> MW

    subgraph HTTP["HTTP adapter"]
        MW["Middleware<br/>security headers · session · requireUser ·<br/>requirePasswordChange · noCache"]
        MW --> Gen["Generated OpenAPI router"]
        Gen --> Handlers["Handlers"]
        MW --> OIDCR["OIDC routes<br/>/api/auth/oidc/login · /callback"]
    end

    subgraph App["Application"]
        Services["Services<br/>account · travel · photo · admin · OIDC"]
        Ports{{"Ports<br/>repositories · storage"}}
        Services --> Ports
    end

    Handlers --> Services
    OIDCR --> Services
    Services --> Domain["Domain<br/>user · country · flight · photo · visibility"]

    subgraph Adapters["Driven adapters"]
        SQLite[("SQLite<br/>travel-blog.db · config.db")]
        Files[("Local photo storage")]
    end

    Ports -. implemented by .-> SQLite
    Ports -. implemented by .-> Files
    OIDCR <-- "OpenID Connect" --> IdP["🔑 Identity provider"]
```

## Authentication

Every request carries an opaque `session` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`, path `/api`). Only the SHA-256 hash of the token is stored, so a leaked database cannot be replayed as a login.

### Password login

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant F as Frontend
    participant B as Backend
    participant D as SQLite

    U->>F: username + password
    F->>B: POST /api/auth/login
    B->>D: load credentials
    B->>B: bcrypt compare<br/>(dummy hash if user unknown)
    alt valid
        B->>D: store session (token hash, expiry)
        B-->>F: 200 + Set-Cookie: session=token
    else invalid
        B-->>F: 401 wrong username or password
    end
    F->>B: GET /api/... (cookie)
    B->>D: look up hash(token), check expiry
    B-->>F: user data
    U->>F: log out
    F->>B: POST /api/auth/logout
    B->>D: delete session
    B-->>F: clear cookie
```

### Single sign-on (OpenID Connect)

Authorization code flow with PKCE (S256), `state` and `nonce`. The flow state lives in a short-lived, HMAC-signed cookie that is valid only once.

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant F as Frontend
    participant B as Backend
    participant P as Identity provider
    participant D as SQLite

    U->>F: click "Log in with …"
    F->>B: GET /api/auth/oidc/login?next=/path
    B->>B: create state, nonce, PKCE verifier
    B-->>U: 302 to provider + signed flow cookie
    U->>P: authenticate
    P-->>U: 302 to /api/auth/oidc/callback?code&state
    U->>B: callback (flow cookie)
    B->>B: verify cookie signature, expiry, state
    B->>P: exchange code + PKCE verifier
    P-->>B: ID token
    B->>B: verify signature, issuer, nonce
    B->>D: find identity (issuer + subject)
    alt known identity
        D-->>B: existing user
    else email verified and matches an account
        B->>D: link identity to that account
    else registration open
        B->>D: create user without password + identity
    else
        B-->>U: 302 /login?error=…
    end
    B->>D: store session (token hash, expiry)
    B-->>U: 302 to next + Set-Cookie: session=token
```

### Request authorization

```mermaid
flowchart TD
    Req["Request"] --> Cookie{"session cookie<br/>valid?"}
    Cookie -- yes --> User["User in context"]
    Cookie -- no --> Dev{"-dev mode?"}
    Dev -- yes --> Admin["Act as passwordless admin"]
    Dev -- no --> Anon["Anonymous"]
    User --> Open
    Admin --> Open
    Anon --> Open{"/api/auth/* or<br/>/api/public/*?"}
    Open -- yes --> Serve["Handle request"]
    Open -- no --> Has{"Has a user?"}
    Has -- no --> E401["401 login required"]
    Has -- yes --> Must{"Must change<br/>password?"}
    Must -- no --> Serve
    Must -- yes --> Only{"GET /api/user or<br/>PUT /api/user/password?"}
    Only -- yes --> Serve
    Only -- no --> E403["403 password change required"]
```
