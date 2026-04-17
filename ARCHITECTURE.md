# ARCHITECTURE.md

## Overview

SimpleNotesServer is a Go service that exposes REST endpoints through Echo, persists application state in SQLite through GORM, and uses AWS-backed services for authentication, file storage, and websocket delivery.

## Entrypoints

Main API process:

- [cmd/api/main.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/api/main.go)

Websocket API Gateway shims:

- [infrastructure/aws/lambda/ws-connect-shim/index.mjs](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/infrastructure/aws/lambda/ws-connect-shim/index.mjs)
- [infrastructure/aws/lambda/ws-message-shim/index.mjs](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/infrastructure/aws/lambda/ws-message-shim/index.mjs)

## Startup Flow

The main process does the following:

1. Loads environment variables from `.env` or AWS SSM.
2. Initializes SQLite and runs `AutoMigrate`.
3. Initializes Cognito, S3, websocket gateway, and the company lookup client.
4. Wires repositories, policies, services, handlers, and middleware.
5. Starts two background jobs:
   - stale websocket connection cleanup
   - expired company cache cleanup
6. Starts the Echo HTTP server on port `7070`.

## Persistence Model

SQLite initialization lives in [db.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/db.go).

Persisted entities:

- `users`
- `notes`
- `connections`
- `companies`
- `company_partners`

The code constrains SQLite to a single open connection, which means query efficiency matters because there is limited room to hide slow scans behind parallelism.

## Request Flow

HTTP route handlers live under [cmd/internal/http/handler](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/http/handler).

Middleware:

- [auth_middleware.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/http/middleware/auth_middleware.go) resolves the authenticated user by `sub_uuid`.

Service layer:

- [user_service.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/service/user_service.go)
- [note_service.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/service/note_service.go)
- [websocket_service.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/service/websocket_service.go)
- [misc_service.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/service/misc_service.go)

Repository layer:

- [cmd/internal/domain/sqlite/repository](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/repository)

## Index-Relevant Paths

These are the main query families that matter for performance:

- user authentication and user existence checks
- websocket presence lookups and fanout by `user_id`
- stale connection cleanup by heartbeat or expiry cutoff
- company cache lookups by `cnpj`
- company cache sweeps by `cached_at`
- note listing filtered by visibility

For index-specific guidance, use [AGENTS.md](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/AGENTS.md).
