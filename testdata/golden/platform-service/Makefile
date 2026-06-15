# Makefile for testproj

run:
	go run cmd/main.go

build:
	go build -o bin/testproj cmd/main.go

migrate-up:
	go run db/migrate.go $(DB) up

migrate-down:
	go run db/migrate.go $(DB) down

migrate-reset:
	go run db/migrate.go $(DB) reset
