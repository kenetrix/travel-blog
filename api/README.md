# API contract

`openapi.yaml` and the files it references are the single source of truth.
Never edit generated files.

| Path                  | Content                                                                        |
| --------------------- | ------------------------------------------------------------------------------ |
| `openapi.yaml`        | Root: info, tags, security, path list, component registry                      |
| `paths/`              | One file per path, named after it (`/user/profile` → `user_profile.yaml`)          |
| `components/schemas/` | Schemas grouped by domain (`common`, `country`, `flight`, `trip`, `account`, `travel`) |
| `components/*.yaml`   | Shared parameters, headers and responses                                       |

New schemas, parameters, headers and responses go into their file under
`components/` and must also be registered by name under `components` in
`openapi.yaml`; otherwise the generated types get other names.

| Target   | Command (run inside the project)                                         | Output                                                  |
| -------- | ------------------------------------------------------------------------ | ------------------------------------------------------- |
| All      | `npm run generate:api` (repo root)                                       | both below                                              |
| Go       | `./scripts/generate-api.sh` or `.\scripts\generate-api.ps1` (`backend/`) | `internal/adapter/http/api/api.gen.go`                  |
| Frontend | `npm run generate:api` (`frontend/`)                                     | `generated/schema.gen.ts` (use via `src/api/client.ts`) |

Both regenerate from scratch; run them after every change to the contract and
commit the generated files together with it. Lint the contract with
`npx redocly lint api/openapi.yaml`.

Go: `oapi-codegen` cannot follow references into other files, so `go generate`
first bundles the contract with Redocly into a temporary, gitignored
`openapi.bundle.yaml` next to the generated code. It needs `npm install` at the
repo root. It generates a strict std-http server. Implement `api.StrictServerInterface`
in `internal/adapter/http/handler`, delegating to the application services.
