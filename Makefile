.PHONY: help start stop restart status logs install-mod build-app got gotc gol golf migrate-up migrate-down mysql clean rebuild restart-app

# Default target
help:
	@./docker/docker-dev.sh help

# Environment Management
start:
	@./docker/docker-dev.sh start

stop:
	@./docker/docker-dev.sh stop

restart:
	@./docker/docker-dev.sh restart

status:
	@./docker/docker-dev.sh status

logs:
	@./docker/docker-dev.sh logs

logs-app:
	@./docker/docker-dev.sh logs app

logs-mysql:
	@./docker/docker-dev.sh logs mysql

# Development (same names as devenv.nix)
install-mod:
	@./docker/docker-dev.sh install-mod

build-app:
	@./docker/docker-dev.sh build-app

start-app:
	@./docker/docker-dev.sh run-app

# Testing & Linting (same names as devenv.nix)
got:
	@./docker/docker-dev.sh got

gotc:
	@./docker/docker-dev.sh gotc

gol:
	@./docker/docker-dev.sh gol

golf:
	@./docker/docker-dev.sh golf

# Database
migrate-create:
	@./docker/docker-dev.sh migrate-create $(name)

migrate-up:
	@./docker/docker-dev.sh migrate-up

migrate-down:
	@./docker/docker-dev.sh migrate-down

mysql-shell:
	@./docker/docker-dev.sh mysql-shell

# Maintenance
clean:
	@./docker/docker-dev.sh clean

rebuild:
	@./docker/docker-dev.sh rebuild

restart-app:
	@./docker/docker-dev.sh restart-app

# Shell access
shell:
	@./docker/docker-dev.sh exec bash
