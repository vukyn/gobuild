#!make
include ./.env
export $(shell sed 's/=.*//' ./.env)

PRJ=

.PHONY: build install version tag

build:
	@echo "Building $(PRJ)..."
	@go build -o bin/ ./$(PRJ)
	@echo "Build complete"

install:
	@echo "Installing $(PRJ)..."
	@go install ./$(PRJ)
	@echo "Install complete"

version:
	@git describe --tags --always --dirty

tag:
	@[ -n "$(VERSION)" ] || { echo "Usage: make tag VERSION=x.y.z"; exit 1; }
	git tag -a v$(VERSION) -m "Release version $(VERSION)"
	git push origin v$(VERSION)