.PHONY: help build build-wasm vet test clean

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
	@python3 scripts/serve.py example/$*/web $(PORT)

clean: ## Remove built WASM binaries and web assets
	rm -f example/*/web/main.wasm example/*/web/wasm_exec.js
