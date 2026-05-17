# Project Structure

This document explains the purpose of each tracked project file. It is meant to
help contributors understand where behavior lives before changing code.

## Root Files

| File | Purpose |
| --- | --- |
| `.dockerignore` | Excludes local-only files, secrets, storage data, and build output from Docker build contexts. |
| `.env.example` | Shows the required local environment variables for the server and CLI. |
| `.gitignore` | Excludes local binaries, test artifacts, editor files, secrets, and runtime storage from Git. |
| `Dockerfile` | Builds the server binary in a Go builder stage and copies it into a small Alpine runtime image. |
| `LICENSE` | Contains the project license. |
| `Makefile` | Provides repeatable development commands for PostgreSQL, migrations, builds, and installation. |
| `Modelfile` | Defines the local LLM prompt used to generate short technical summaries for stored notes. |
| `README.md` | Main project documentation, setup guide, API reference, and development reference. |
| `docker-compose.yml` | Runs the local PostgreSQL database used during development. |
| `go.mod` | Declares the Go module path, Go version, and direct/indirect dependencies. |
| `go.sum` | Locks dependency checksums for reproducible module downloads. |
| `main` | Local build artifact currently tracked in the repository; it is not part of the application source. |
| `sqlc.yaml` | Configures sqlc to generate Go database code from migrations and query files. |

## CI/CD

| File | Purpose |
| --- | --- |
| `.woodpecker/lint.yaml` | Runs formatting and static analysis checks on push events. |
| `.woodpecker/test.yaml` | Runs Go tests with coverage on push events. |
| `.woodpecker/security.yaml` | Runs gosec security scanning on push events. |
| `.woodpecker/build.yaml` | Builds and pushes Docker images for `main` branch pushes after checks pass. |
| `.woodpecker/deploy.yaml` | Applies Kubernetes manifests and rolls out the image built from the current commit. |

## Commands

| File | Purpose |
| --- | --- |
| `cmd/server/main.go` | Server entrypoint. It loads configuration, opens the database pool, creates storage, registers HTTP handlers, and performs graceful shutdown. |
| `cmd/recall/main.go` | CLI entrypoint. It delegates command execution to the Cobra command tree. |
| `cmd/recall/cmd/root.go` | Defines the root CLI command, global flags, and configuration loading. |
| `cmd/recall/cmd/init.go` | Implements `recall init`, which writes the user-level CLI config file. |
| `cmd/recall/cmd/upload.go` | Implements `recall upload`, including title/tag handling and multipart upload. |
| `cmd/recall/cmd/list.go` | Implements `recall list`, including query/all validation and terminal result selection. |
| `cmd/recall/cmd/get.go` | Implements `recall get`, including document selection, content download, and Markdown rendering. |
| `cmd/recall/cmd/helpers.go` | Contains shared CLI helpers for styled errors, missing-content messages, and Markdown detection. |

## Internal Packages

| File | Purpose |
| --- | --- |
| `internal/api/doc.go` | Package-level documentation for shared API contracts. |
| `internal/api/types.go` | Defines JSON response structs shared by the server and CLI client. |
| `internal/client/doc.go` | Package-level documentation for the CLI HTTP client. |
| `internal/client/client.go` | Implements the authenticated HTTP client for listing, uploading, and downloading documents. |
| `internal/client/client_test.go` | Tests client construction and URL validation behavior. |
| `internal/config/doc.go` | Package-level documentation for server configuration loading. |
| `internal/config/config.go` | Loads server configuration from environment variables and validates required values. |
| `internal/handlers/doc.go` | Package-level documentation for HTTP handler responsibilities. |
| `internal/handlers/handlers.go` | Assembles handler dependencies, routes, and middleware into an `http.Handler`. |
| `internal/handlers/routes.go` | Registers the HTTP routes served by the API. |
| `internal/handlers/middleware.go` | Provides bearer-token authentication middleware. |
| `internal/handlers/response.go` | Provides JSON response and error helpers. |
| `internal/handlers/health.go` | Implements the unauthenticated health endpoint. |
| `internal/handlers/documents.go` | Implements document upload, list, metadata lookup, content streaming, slug creation, and tag normalization. |
| `internal/storage/doc.go` | Package-level documentation for filesystem storage. |
| `internal/storage/files.go` | Provides safe local file storage rooted at a configured base directory. |
| `internal/storage/files_test.go` | Tests storage path generation, extension validation, and cleanup after write failures. |

## Database

| File | Purpose |
| --- | --- |
| `internal/storage/database/db.go` | sqlc-generated database interface and transaction helpers. |
| `internal/storage/database/models.go` | sqlc-generated Go model definitions for database rows. |
| `internal/storage/database/documents.sql.go` | sqlc-generated methods for document queries. |
| `internal/storage/database/pool.go` | Hand-written pgx pool constructor with startup ping validation. |
| `internal/storage/queries/documents.sql` | Source SQL query definitions used by sqlc. |
| `internal/storage/schemas/001_create_documents.sql` | Creates the initial `documents` table. |
| `internal/storage/schemas/002_add_tags.sql` | Adds document tags and a GIN index. |
| `internal/storage/schemas/003_change_size_bytes.sql` | Changes `size_bytes` from `INT` to `BIGINT`. |
| `internal/storage/schemas/004_add_deleted_at.sql` | Adds nullable `deleted_at` for soft-delete filtering. |

## Terminal UI

| File | Purpose |
| --- | --- |
| `internal/tui/docglamour/doc.go` | Package-level documentation for Markdown rendering. |
| `internal/tui/docglamour/docglamour.go` | Renders Markdown content inside a Bubble Tea viewport using Glamour. |
| `internal/tui/doclist/doc.go` | Package-level documentation for document result selection. |
| `internal/tui/doclist/doclist.go` | Fetches documents and displays them in an interactive list. |
| `internal/tui/docmultinput/doc.go` | Package-level documentation for the init form. |
| `internal/tui/docmultinput/docmultinput.go` | Displays the server URL/token input form used by `recall init`. |
| `internal/tui/docresult/doc.go` | Package-level documentation for overwrite confirmation. |
| `internal/tui/docresult/docresult.go` | Displays a confirmation prompt when a CLI config file already exists. |

## Deployment

| File | Purpose |
| --- | --- |
| `k8s/postgres.yaml` | Defines PostgreSQL credentials, storage, deployment, and service for Kubernetes. |
| `k8s/recall.yaml` | Defines the recall server deployment, storage mount, health probes, and service. |
