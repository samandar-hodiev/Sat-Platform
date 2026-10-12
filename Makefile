# SAT Platform developer commands. Run `make help` for the list.
# Kept deliberately small: each target is one or two plain commands you can also run by hand.

TEST_DATABASE_URL ?=

.DEFAULT_GOAL := help
.PHONY: help setup api web migrate-up migrate-down migrate-status migrate-create test test-api test-api-integration test-web lint fmt fmt-check typecheck build gitpulse-hook

## help: list available commands
help:
	@grep -E "^## " Makefile | sed "s/^## /  /"

## setup: create .env files from the examples and install Go + npm dependencies
setup:
	@test -f .env || (cp .env.example .env && echo "created .env — set DATABASE_URL")
	@test -f web/.env.local || (cp web/.env.example web/.env.local && echo "created web/.env.local")
	cd api && go mod download
	cd web && npm install

## api: run the Go API (http://localhost:8090)
api:
	cd api && go run ./cmd/server

## web: run the Next.js app (http://localhost:3000)
web:
	cd web && npm run dev

## migrate-up: apply pending database migrations to DATABASE_URL (root .env)
migrate-up:
	cd api && go run ./cmd/migrate up

## migrate-down: revert the last migration (make migrate-down n=3 for more)
migrate-down:
	cd api && go run ./cmd/migrate down $(or $(n),1)

## migrate-status: show the applied and pending migrations
migrate-status:
	cd api && go run ./cmd/migrate status

## migrate-create: new empty migration pair, e.g. make migrate-create name=create_learner_goals
migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=snake_case_name" && exit 1)
	cd api && go run ./cmd/migrate create $(name)

## test: Go unit tests + web unit tests
test: test-api test-web

## test-api: Go unit tests (integration tests skip without TEST_DATABASE_URL)
test-api:
	cd api && go test ./...

## test-api-integration: Go tests against a disposable database, e.g. make test-api-integration TEST_DATABASE_URL=postgres://localhost:5432/sat_platform_test
test-api-integration:
	@test -n "$(TEST_DATABASE_URL)" || (echo "set TEST_DATABASE_URL to a disposable database" && exit 1)
	cd api && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -count=1 -p 1 ./...

## test-web: web unit tests (vitest)
test-web:
	cd web && npm test

## lint: gofmt check + go vet + eslint + prettier check
lint: fmt-check
	cd api && go vet ./...
	cd web && npm run lint && npm run format:check

## fmt: format Go (gofmt) and web (prettier) code
fmt:
	cd api && gofmt -w .
	cd web && npm run format

## fmt-check: fail if any Go file is not gofmt-formatted
fmt-check:
	@cd api && files=$$(gofmt -l .); \
		if [ -n "$$files" ]; then echo "not gofmt-formatted (run make fmt):"; echo "$$files"; exit 1; fi

## typecheck: TypeScript check for the web app
typecheck:
	cd web && npm run typecheck

## build: compile the API binary (api/bin/server) and build the web app
build:
	cd api && go build -o bin/server ./cmd/server
	cd web && npm run build

## gitpulse-hook: install the pre-push hook that announces pushes in Telegram via GitPulse
gitpulse-hook:
	@cp infrastructure/scripts/pre-push-hook.sh .git/hooks/pre-push && chmod +x .git/hooks/pre-push && echo "pre-push hook installed"
