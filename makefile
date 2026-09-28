FRONTEND_DIR = ./web
BACKEND_DIR = .

.PHONY: all build-frontend start-backend

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

# Project-local decision notes; no npm install or network needed.
.PHONY: verify-notes
verify-notes:
	bun .agents/skills/write-notes-like-deepseek/scripts/verify-agent-note-tree.ts
	bun .agents/skills/write-notes-like-deepseek/scripts/verify-agent-note-format.ts
	bun .agents/skills/write-notes-like-deepseek/scripts/verify-archived-agent-notes.ts
