#!make
include ./.env
export $(shell sed 's/=.*//' ./.env)

PRJ=

build:
	@echo "Building $(PRJ)..."
	@go build -o bin/ ./$(PRJ)
	@echo "Build complete"

install:
	@echo "Installing $(PRJ)..."
	@go install ./$(PRJ)
	@echo "Install complete"

tag:
	git tag -a v$(VERSION) -m "Release version $(VERSION)"
	git push origin v$(VERSION)