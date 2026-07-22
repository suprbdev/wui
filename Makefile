.PHONY: help build build-wasm vet test clean release

EXAMPLES := counter form todo timer
GOROOT   := $(shell go env GOROOT)
WASM_EXEC := $(GOROOT)/lib/wasm/wasm_exec.js
PORT     ?=

help: ## Show this help
	@grep -E '^[a-zA-Z_%-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo "  examples: $(EXAMPLES)"

build: ## Build the wui package (TUI)
	go build ./...

build-wasm: ## Build the wui package (WASM)
	GOOS=js GOARCH=wasm go build ./...

vet: ## Run go vet on TUI and WASM targets
	go vet ./...
	GOOS=js GOARCH=wasm go vet .

test: ## Run tests
	go test ./...

run-%: ## Run an example in the terminal (e.g. make run-counter)
	go run ./example/$*

tui-%: wasm-% ## Run an example in the terminal + serve its web build (e.g. make tui-counter)
	go run ./example/$* -serve

wasm-%: ## Build an example as WASM into example/NAME/web/ (e.g. make wasm-counter)
	mkdir -p example/$*/web
	GOOS=js GOARCH=wasm go build -o example/$*/web/main.wasm ./example/$*
	cp $(WASM_EXEC) example/$*/web/wasm_exec.js

serve-%: wasm-% ## Build and serve an example web build; auto-picks a free port (override: PORT=8765)
	@go run ./cmd/serve example/$*/web $(PORT)

clean: ## Remove built WASM binaries and web assets
	rm -f example/*/web/main.wasm example/*/web/wasm_exec.js

# Verifies, tags, and pushes; the Release workflow (release.yaml) then
# publishes the GitHub release with generated notes.
release: ## Cut a new release (prompts for version, tags, pushes)
	@set -e; \
	git diff --quiet && git diff --cached --quiet || { echo "error: working tree dirty — commit or stash first"; exit 1; }; \
	current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "(none)"); \
	echo "Current version: $$current"; \
	printf "New version (vX.Y.Z): "; read -r version; \
	echo "$$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$$' || { echo "error: invalid version '$$version' (expected vX.Y.Z)"; exit 1; }; \
	if git rev-parse -q --verify "refs/tags/$$version" >/dev/null; then echo "error: tag $$version already exists"; exit 1; fi; \
	$(MAKE) vet test; \
	git tag -a "$$version" -m "$$version"; \
	git push origin HEAD "$$version"; \
	url=$$(git remote get-url origin | sed -E 's#^git@github\.com:#https://github.com/#; s#\.git$$##'); \
	echo "Pushed $$version — the Release workflow is publishing it:"; \
	echo "  $$url/actions/workflows/release.yaml"
