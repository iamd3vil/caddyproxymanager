# Backend commands
backend-run:
    cd backend && go run ./cmd/server

backend-build:
    cd backend && go build -o ../bin/caddyproxymanager ./cmd/server

backend-test:
    cd backend && go test ./...

backend-fmt:
    cd backend && go fmt ./...

backend-vet:
    cd backend && go vet ./...

backend-tidy:
    cd backend && go mod tidy

# Frontend commands
frontend-install:
    cd frontend && npm install

frontend-dev:
    cd frontend && npm run dev

frontend-build:
    cd frontend && npm run build

frontend-lint:
    cd frontend && npm run lint

frontend-type-check:
    cd frontend && npm run type-check

# Development
dev: backend-run

# Build everything
build: backend-build frontend-build

# Setup project
setup: backend-tidy frontend-install

# Clean
clean:
    rm -rf bin/
    rm -rf frontend/dist/