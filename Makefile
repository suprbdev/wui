.PHONY: help build build-wasm vet test run-counter run-form wasm-counter wasm-form serve-counter serve-form clean

GOROOT := $(shell go env GOROOT)
WASM_EXEC := $(GOROOT)/lib/wasm/wasm_exec.js

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the wui package (TUI)
	go build ./...

build-wasm: ## Build the wui package (WASM)
	GOOS=js GOARCH=wasm go build ./...

vet: ## Run go vet on TUI and WASM targets
	go vet ./...
	GOOS=js GOARCH=wasm go vet .

test: ## Run tests
	go test ./...

run-counter: ## Run the counter example in the terminal
	go run ./example/counter

run-form: ## Run the form example in the terminal
	go run ./example/form

wasm-counter: ## Build the counter example as WASM into example/counter/web/
	mkdir -p example/counter/web
	GOOS=js GOARCH=wasm go build -o example/counter/web/main.wasm ./example/counter
	cp $(WASM_EXEC) example/counter/web/wasm_exec.js

wasm-form: ## Build the form example as WASM into example/form/web/
	mkdir -p example/form/web
	GOOS=js GOARCH=wasm go build -o example/form/web/main.wasm ./example/form
	cp $(WASM_EXEC) example/form/web/wasm_exec.js

serve-counter: wasm-counter ## Build and serve the counter WASM example on :8765
	@echo "Serving counter at http://localhost:8765"
	@python3 -c "\
import http.server, mimetypes, os, sys; \
mimetypes.add_type('application/wasm', '.wasm'); \
os.chdir('example/counter/web'); \
http.server.test(HandlerClass=http.server.SimpleHTTPRequestHandler, port=8765)"

serve-form: wasm-form ## Build and serve the form WASM example on :8765
	@echo "Serving form at http://localhost:8765"
	@python3 -c "\
import http.server, mimetypes, os, sys; \
mimetypes.add_type('application/wasm', '.wasm'); \
os.chdir('example/form/web'); \
http.server.test(HandlerClass=http.server.SimpleHTTPRequestHandler, port=8765)"

clean: ## Remove built WASM binaries and web assets
	rm -f example/counter/web/main.wasm example/counter/web/wasm_exec.js
	rm -f example/form/web/main.wasm example/form/web/wasm_exec.js
