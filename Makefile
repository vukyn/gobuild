#!make
# ⚠️ `-include`, not `include`, and the sed is guarded. .env is gitignored, so a
# fresh clone does not have one — with the bare forms every target failed with
# "No such file or directory. Stop." before reaching a recipe, which made the
# Makefile unusable anywhere but a developer's own checkout. Only `tag` actually
# reads a value out of .env, and it already errors on its own when VERSION is
# unset.
-include ./.env
export $(shell [ -f ./.env ] && sed 's/=.*//' ./.env)

PRJ=

.PHONY: build install version tag check scan-generated

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

# The fast gate: everything that needs no network and runs in seconds.
# ⚠️ `gofmt -l` prints its complaints and exits 0, so the bare form would report
# unformatted files and still pass. The -z test is what makes it a gate.
check:
	@echo "==> gofmt"
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt: files need formatting"; exit 1; }
	@echo "==> go build"
	@go build ./...
	@echo "==> go vet"
	@go vet ./...
	@echo "==> go test"
	@go test ./... -count=1

# The slow gate: scan the code this tool EMITS rather than the code it is.
#
# `make check` covers gobuild's own ~600 lines. The thousands of lines that
# reach real services live under templates/ as .tmpl, which no scanner can
# parse — gosec reporting "0 issues" here has never once looked at them. This
# renders each preset into a throwaway directory and scans it there.
#
# Needs network (the scaffold runs `go mod tidy`; npm audit needs a resolved
# lockfile). ~17s warm, ~46s cold. Scan one preset with:
#   make scan-generated PRESETS=platform-service
scan-generated:
	@./scripts/scan-generated.sh $(PRESETS)
