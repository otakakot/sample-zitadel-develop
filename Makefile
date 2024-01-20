SHELL := /bin/bash
include .env
export
export APP_NAME := $(basename $(notdir $(shell pwd)))

.PHONY: help
help: ## display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: mod
mod: ## go modules
	@go get -u -t ./...
	@go mod tidy
	@go mod vendor

.PHONY: tool
tool: ## install tool
	@aqua install

.PHONY: user
user: ## create user
	@go run cmd/user/main.go

.PHONY: oidcrp
oidcrp: ## create oidc rp
	@go run cmd/oidcrp/main.go

.PHONY: up
up: ## docker compose up with air hot reload
	@mkdir -p machinekey
	@docker compose --project-name ${APP_NAME} --file ./.docker/compose.yaml up -d

.PHONY: down
down: ## docker compose down
	@docker compose --project-name ${APP_NAME} down --volumes

.PHONY: balus
balus: ## destroy everything about docker. (containers, images, volumes, networks.)
	@docker compose --project-name ${APP_NAME} down --rmi all --volumes
