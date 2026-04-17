# AGENTS.md

## Goal

This repository is a Go API backed by SQLite and exposed through Echo HTTP routes plus AWS API Gateway websocket shims. When working on index optimization here, optimize for the queries the application actually runs, not hypothetical future dashboards from the land of "maybe someday".

## Scope

This document is for agents making schema, query, or migration changes related to database indexes.

Acceptance criteria for index work:

- Every proposed index is tied to a concrete repository query or job.
- Index changes are small and reviewable.
- Query behavior is validated with SQLite tools such as `EXPLAIN QUERY PLAN`.
- Docs are updated when query shapes or index strategy changes.
- If behavior changes, tests are added or updated.

## Runtime Entrypoints

Primary server entrypoint:

- [cmd/api/main.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/api/main.go)

Supporting websocket entrypoints:

- [infrastructure/aws/lambda/ws-connect-shim/index.mjs](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/infrastructure/aws/lambda/ws-connect-shim/index.mjs)
- [infrastructure/aws/lambda/ws-message-shim/index.mjs](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/infrastructure/aws/lambda/ws-message-shim/index.mjs)

SQLite bootstrap and automigration:

- [cmd/internal/domain/sqlite/db.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/db.go)

## Data Access Hot Paths

Read these before touching indexes:

- [cmd/internal/domain/sqlite/repository/user_repository.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/repository/user_repository.go)
- [cmd/internal/domain/sqlite/repository/connection_repository.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/repository/connection_repository.go)
- [cmd/internal/domain/sqlite/repository/company_repository.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/repository/company_repository.go)
- [cmd/internal/domain/sqlite/repository/note_repository.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/repository/note_repository.go)

Current high-value query patterns:

- `users.email + active`
  Used by login, signup checks, confirmation, resend confirmation.
- `users.sub_uuid + active`
  Used by auth middleware on protected routes and websocket connect flow.
- `users.id + active`
  Used when resolving active users by route params or event recipients.
- `connections.user_id`
  Used for fanout, connection termination, online checks, and counting user connections.
- `connections.connection_id`
  Primary key lookup for connect/disconnect/message handling.
- `connections.expires_at` and `connections.last_heartbeat_at`
  Used by the background stale-connection cleaner.
- `companies.cnpj`
  Used for company cache lookup.
- `company_partners.company_cnpj`
  Used by company preload.
- `companies.cached_at`
  Used by the cache cleanup job.
- `notes.visibility`
  Used by public note listing when private notes must be excluded.

## Current Index State

Indexes already declared in entities:

- `connections.user_id`
- `connections.last_heartbeat_at`
- `company_partners(company_cnpj, name)` as a unique composite index

Noticeable gaps between query patterns and declared indexes:

- `users.email + active`
- `users.sub_uuid + active`
- `companies.cached_at`
- `notes.visibility`
- `connections.expires_at`

## Index Strategy For This Repo

Prefer indexes that match concrete filters in the repositories and middleware.

Good candidates based on current code:

- `users(active, email)` or `users(email, active)`
  Needed for repeated active-email lookups. In SQLite, equality predicates on both columns can use either order, but prefer the order that matches the most selective column in real data.
- `users(active, sub_uuid)` or `users(sub_uuid, active)`
  Critical for auth middleware and websocket-authenticated flows.
- `users(active, id)`
  Usually unnecessary because `id` is already the primary key. Do not add this unless query plans prove it matters.
- `connections(user_id)`
  Already present and should stay.
- `connections(expires_at)`
  Helps the stale-session cleanup branch.
- `connections(last_heartbeat_at)`
  Already present and should stay.
- `companies(cached_at)`
  Helps hourly TTL cleanup avoid full scans.
- `notes(visibility)`
  Reasonable only if note volume is large enough for the public-notes query to stop being a table scan bottleneck.

Be skeptical of these:

- Indexing `notes.tags`
  Tags are stored as space-delimited text, which is not index-friendly for real search.
- Adding broad multi-column indexes "just in case"
  Every extra index increases write cost and migration complexity.
- Duplicating primary key coverage
  `users.id`, `notes.id`, `connections.connection_id`, and `companies.cnpj` already have primary key indexes.

## Decision Rules

Before adding an index:

1. Find the exact repository or middleware query that needs it.
2. Check whether an existing primary or secondary index already covers it.
3. Inspect the table cardinality and the query frequency.
4. Use `EXPLAIN QUERY PLAN` on the real SQL shape.
5. Prefer one well-targeted composite index over multiple overlapping single-column indexes when the app always filters on the same column pair.

Do not add an index when:

- The table is tiny and remains tiny.
- The query is write-heavy and the read path is not hot.
- The filter is low-selectivity and rarely used.
- The apparent problem is really an N+1 pattern or an avoidable full-table read in service code.

## Validation Workflow

When changing indexes, validate using the actual SQLite database and realistic predicates.

Useful checks:

```powershell
sqlite3 database.db "EXPLAIN QUERY PLAN SELECT * FROM users WHERE email = 'a@b.com' AND active = 1;"
sqlite3 database.db "EXPLAIN QUERY PLAN SELECT * FROM users WHERE sub_uuid = 'subject' AND active = 1;"
sqlite3 database.db "EXPLAIN QUERY PLAN SELECT * FROM connections WHERE expires_at < 1234567890 OR last_heartbeat_at < 1234567890;"
sqlite3 database.db "EXPLAIN QUERY PLAN SELECT * FROM companies WHERE cached_at < 1234567890;"
```

Also inspect the generated schema:

```powershell
sqlite3 database.db ".schema users"
sqlite3 database.db ".schema connections"
sqlite3 database.db ".schema companies"
sqlite3 database.db ".schema notes"
```

If SQLite CLI is unavailable, use a Go test or one-off validation path that prints the query plan from the same schema the app uses.

## GORM Notes

- Indexes can be declared with `gorm` tags in entity structs under `cmd/internal/domain/entity`.
- `AutoMigrate` in [db.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/db.go) is responsible for schema alignment, so index declarations must be safe to apply to existing local databases.
- Be cautious with renaming or replacing indexes. `AutoMigrate` is not a magical schema wizard; sometimes it is more of a polite suggestion engine.

## File Map For Index Work

Likely edit points:

- [cmd/internal/domain/entity/user.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/entity/user.go)
- [cmd/internal/domain/entity/connection.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/entity/connection.go)
- [cmd/internal/domain/entity/company.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/entity/company.go)
- [cmd/internal/domain/entity/note.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/entity/note.go)
- [cmd/internal/domain/sqlite/repository](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/repository)
- [cmd/internal/domain/sqlite/db.go](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/cmd/internal/domain/sqlite/db.go)

## Minimum Review Checklist

- Does the index match a real query in current code?
- Is the column order justified?
- Are we accidentally duplicating primary-key coverage?
- Does the change improve `EXPLAIN QUERY PLAN` for the target query?
- Are writes or cleanup jobs made meaningfully more expensive?
- Were docs and tests updated if behavior changed?

## Related Documentation

- [ARCHITECTURE.md](C:/Users/Leonardo/Documents/Repositories/Magalu/SimpleNotesServer/ARCHITECTURE.md)
