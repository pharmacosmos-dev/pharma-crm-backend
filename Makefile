CURRENT_DIR=$(shell pwd)

-include .env

LOCAL_BIN:=$(CURDIR)/bin
PATH:=$(LOCAL_BIN):$(PATH)

DB_URL="postgres://$(PG_USER):$(PG_PASS)@$(PG_HOST):$(PG_PORT)/$(PG_DB)?sslmode=disable"
# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

run:
	go run cmd/app/main.go

swaggo:
	swag init -g internal/controller/http/router.go -o docs

migrate_file:  ### create new migration
	./scripts/migrate.sh
#	migrate create -ext sql -dir migrations 'table_name'
.PHONY: migrate_file

migrate_up: ### migration up
	migrate -path migrations -database "$(DB_URL)" up

.PHONY: migrate_up

migrate_down:
	migrate -path migrations -database "$(DB_URL)" down
.PHONY: migrate_down

migrate_force:
	migrate -path migrations -database "$(DB_URL)" force 18
.PHONY: migrate_force