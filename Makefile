BINARY ?= daylist
GO ?= go
COVERAGE ?= coverage.out
COVERAGE_HTML ?= coverage.html

.DEFAULT_GOAL := help
.PHONY: help build install test test-race vet fmt fmt-check lint cover ci clean

help:
	@echo "Uso: make <alvo>"
	@echo ""
	@echo "  build       Gera ./$(BINARY)"
	@echo "  install     Instala com go install ."
	@echo "  test        Roda go test ./..."
	@echo "  test-race   Roda go test -race -cover ./..."
	@echo "  vet         Roda go vet ./..."
	@echo "  fmt         Formata os arquivos Go com gofmt"
	@echo "  fmt-check   Falha se houver arquivo fora do gofmt"
	@echo "  lint        Executa fmt-check e vet"
	@echo "  cover       Gera $(COVERAGE) e $(COVERAGE_HTML)"
	@echo "  ci          Executa fmt-check, vet, test-race e build"
	@echo "  clean       Remove binário e artefatos de cobertura"

build:
	$(GO) build -o $(BINARY) .

install:
	$(GO) install .

test:
	$(GO) test ./...

test-race:
	$(GO) test -race -cover ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "Arquivos fora do gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi

lint: fmt-check vet

cover:
	$(GO) test -coverprofile=$(COVERAGE) ./...
	$(GO) tool cover -html=$(COVERAGE) -o $(COVERAGE_HTML)
	@echo "Cobertura: $(COVERAGE_HTML)"

ci: fmt-check vet test-race build

clean:
	rm -f $(BINARY) $(COVERAGE) $(COVERAGE_HTML)
