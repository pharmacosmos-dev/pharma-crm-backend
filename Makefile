CURRENT_DIR=$(shell pwd)

-include .env


LOCAL_BIN:=$(CURDIR)/bin
PATH:=$(LOCAL_BIN):$(PATH)

# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

run:
	go run cmd/app/main.go

swaggo:
	echo "Starting swagger generating"
	swag init -g internal/controller/http/v1/router.go -o docs

migrate-create:  ### create new migration
	./scripts/migrate.sh
#	migrate create -ext sql -dir migrations 'table_name'
.PHONY: migrate-create

migrate-up: ### migration up
	migrate -path migrations -database '$(PG_URL)?sslmode=disable' up
.PHONY: migrate-up