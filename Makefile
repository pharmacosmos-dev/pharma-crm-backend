CURRENT_DIR=$(shell pwd)

-include .env

run:
	go run cmd/app/main.go

swaggo:
	echo "Starting swagger generating"
	swag init -g internal/controller/http/v1/router.go -o docs