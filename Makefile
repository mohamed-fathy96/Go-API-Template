GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

SWAG := $(GOBIN)/swag
SWAG_VERSION := v1.16.3

.PHONY: swagger
swagger: $(SWAG)
	$(SWAG) init -g cmd/api/main.go -o docs --parseDependency --parseInternal

$(SWAG):
	@echo "Installing swag CLI ($(SWAG_VERSION))..."
	@go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	@echo "Installed at $(SWAG)"
