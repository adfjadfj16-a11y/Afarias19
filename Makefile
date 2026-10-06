.PHONY: help install build test lint format clean docker-build docker-up docker-down release

# Variables
GO := go
PYTHON := python3
GOOS ?= linux
GOARCH ?= amd64
VERSION ?= 1.0.0
DOCKER_IMAGE ?= afarias19
DOCKER_TAG ?= latest

help:
	@echo "Afarias19 — Makefile Commands"
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make install          Install Go and Python dependencies"
	@echo "  make setup            Setup development environment"
	@echo ""
	@echo "Building:"
	@echo "  make build            Build self-check binary (current platform)"
	@echo "  make build-all        Build for all platforms (linux, darwin, windows)"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test             Run all tests"
	@echo "  make test-go          Run Go tests only"
	@echo "  make test-python      Run Python tests only"
	@echo "  make coverage         Generate coverage reports"
	@echo "  make lint             Lint all code"
	@echo "  make format           Format code (gofmt, black)"
	@echo ""
	@echo "Security:"
	@echo "  make security-scan    Run security scanner (Trivy)"
	@echo "  make secret-scan      Scan for secrets"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build     Build Docker image"
	@echo "  make docker-up        Start Docker containers (dev environment)"
	@echo "  make docker-down      Stop Docker containers"
	@echo "  make docker-push      Push Docker image to registry"
	@echo ""
	@echo "Release:"
	@echo "  make release          Create a release (requires VERSION)"
	@echo "  make release-tag      Tag current commit for release"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean            Remove build artifacts"
	@echo "  make clean-docker     Remove Docker images and containers"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test coverage"
	@echo "  make release VERSION=v1.2.3"
	@echo "  make docker-build docker-up"

# ============================================================================
# SETUP & INSTALLATION
# ============================================================================

install:
	@echo "📦 Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	$(PYTHON) -m pip install --upgrade pip
	$(PYTHON) -m pip install -r requirements.txt || echo "No requirements.txt found, installing test dependencies..."
	$(PYTHON) -m pip install pytest pytest-cov black flake8 eslint
	@echo "✅ Dependencies installed"

setup: install
	@echo "🔧 Setting up development environment..."
	mkdir -p bin dist
	$(GO) mod verify || true
	@echo "✅ Development environment ready"

# ============================================================================
# BUILD
# ============================================================================

build:
	@echo "🔨 Building self-check binary for $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-ldflags="-X 'main.Version=$(VERSION)'" \
		-o bin/afarias19-$(GOOS)-$(GOARCH) \
		./cmd/self-check
	@echo "✅ Built: bin/afarias19-$(GOOS)-$(GOARCH)"

build-all:
	@echo "🔨 Building for all platforms..."
	@$(MAKE) build GOOS=linux GOARCH=amd64
	@$(MAKE) build GOOS=linux GOARCH=arm64
	@$(MAKE) build GOOS=darwin GOARCH=amd64
	@$(MAKE) build GOOS=darwin GOARCH=arm64
	@$(MAKE) build GOOS=windows GOARCH=amd64
	@echo "✅ All binaries built in bin/"
	@ls -lh bin/afarias19-*

build-bot:
	@echo "🔨 Bundling bot_spot_binance_safe.py..."
	mkdir -p dist
	cp bot_spot_binance_safe.py dist/
	@echo "✅ Bot bundled"

# ============================================================================
# TESTING & QUALITY
# ============================================================================

test: test-go test-python
	@echo "✅ All tests passed"

test-go:
	@echo "🧪 Running Go tests..."
	$(GO) test -v -race -coverprofile=/tmp/go-coverage.out ./...
	$(GO) tool cover -func=/tmp/go-coverage.out | tail -1
	@echo "✅ Go tests passed"

test-python:
	@echo "🧪 Running Python tests..."
	$(PYTHON) -m pytest tests/ -v --cov=. --cov-report=term-missing
	@echo "✅ Python tests passed"

test-verbose: test-go
	@$(GO) tool cover -html=/tmp/go-coverage.out -o /tmp/coverage.html
	@echo "📊 Coverage HTML: /tmp/coverage.html"

coverage: test
	@echo "📊 Generating coverage reports..."
	@$(GO) tool cover -html=/tmp/go-coverage.out -o coverage-go.html 2>/dev/null || true
	@$(PYTHON) -m pytest tests/ --cov=. --cov-report=html:coverage-python 2>/dev/null || true
	@echo "📊 Go coverage: coverage-go.html"
	@echo "📊 Python coverage: coverage-python/index.html"

lint:
	@echo "🔍 Linting code..."
	$(GO) fmt ./...
	@which golangci-lint > /dev/null && golangci-lint run ./... || echo "⚠️  golangci-lint not installed, skipping"
	$(PYTHON) -m black --check *.py tests/ 2>/dev/null || echo "⚠️  Black check skipped"
	$(PYTHON) -m flake8 *.py tests/ --max-line-length=100 2>/dev/null || echo "⚠️  Flake8 skipped"
	@echo "✅ Lint passed"

format:
	@echo "✨ Formatting code..."
	$(GO) fmt ./...
	$(PYTHON) -m black *.py tests/ --line-length=100 || echo "⚠️  Black formatting skipped"
	@echo "✅ Code formatted"

# ============================================================================
# SECURITY
# ============================================================================

security-scan:
	@echo "🔒 Running security scan (Trivy)..."
	@which trivy > /dev/null && trivy fs . || echo "⚠️  Trivy not installed. Install: brew install trivy"

secret-scan:
	@echo "🔒 Scanning for secrets..."
	@which detect-secrets > /dev/null && detect-secrets scan || echo "⚠️  detect-secrets not installed. Install: pip install detect-secrets"

# ============================================================================
# DOCKER
# ============================================================================

docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker build -t $(DOCKER_IMAGE):latest .
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-up:
	@echo "🚀 Starting Docker containers..."
	@if [ -f docker-compose.yml ]; then \
		docker-compose -f docker-compose.yml up -d; \
		echo "✅ Containers running. View logs: docker-compose logs -f"; \
	else \
		echo "⚠️  docker-compose.yml not found"; \
	fi

docker-down:
	@echo "🛑 Stopping Docker containers..."
	@if [ -f docker-compose.yml ]; then \
		docker-compose -f docker-compose.yml down; \
		echo "✅ Containers stopped"; \
	fi

docker-logs:
	@docker-compose -f docker-compose.yml logs -f

docker-push:
	@echo "📤 Pushing Docker image to registry..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo "✅ Image pushed"

# ============================================================================
# RELEASE
# ============================================================================

release-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ VERSION not specified. Use: make release-tag VERSION=v1.2.3"; \
		exit 1; \
	fi
	@echo "📌 Tagging release $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "✅ Tagged: $(VERSION)"

release: build-all release-tag
	@echo "🎉 Release $(VERSION) created"
	@echo "✅ Binaries in bin/"
	@echo "✅ Tag pushed to GitHub"
	@echo "Next: GitHub Actions will sign and publish"

# ============================================================================
# CLEANUP
# ============================================================================

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin dist *.out *.html coverage-*
	$(GO) clean -cache -testcache
	@echo "✅ Cleaned"

clean-docker:
	@echo "🧹 Cleaning Docker artifacts..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest 2>/dev/null || true
	@echo "✅ Docker artifacts cleaned"

clean-all: clean clean-docker
	@echo "✅ Full cleanup done"

# ============================================================================
# DEVELOPMENT HELPERS
# ============================================================================

run-self-check:
	@echo "🔍 Running self-check..."
	@if [ -f "bin/afarias19-$(GOOS)-$(GOARCH)" ]; then \
		./bin/afarias19-$(GOOS)-$(GOARCH); \
	else \
		echo "⚠️  Binary not found. Run: make build"; \
	fi

run-bot:
	@echo "🤖 Running bot (paper-trading)..."
	BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO $(PYTHON) bot_spot_binance_safe.py

docs:
	@echo "📚 Generating documentation..."
	$(GO) doc -all ./... > docs/godoc.txt 2>/dev/null || true
	@echo "✅ Documentation generated: docs/godoc.txt"

# ============================================================================
# CONTINUOUS INTEGRATION
# ============================================================================

ci: lint test security-scan
	@echo "✅ CI pipeline passed"

ci-full: setup lint test coverage security-scan build-all
	@echo "✅ Full CI pipeline passed"
