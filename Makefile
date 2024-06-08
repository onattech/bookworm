# Include variables from the .envrc file
include .envrc

# List command to list all commands. 
.PHONY: list
list:
	@echo "📋 Available commands:"
	@awk -F':.*?## ' '/^[a-zA-Z0-9_/-]+:/ && !/^[[:blank:]]*list|confirm:/ { if ($$2 == "") { printf "   • %s\n", $$1 } else { printf "   • %-20s %s\n", $$1, $$2 } }' $(MAKEFILE_LIST)

# Reusable confirm command. It won't get listed
.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]
	
# ══════════════════════════════════════════
#                 DEVELOPMENT               
# ══════════════════════════════════════════

.PHONY: run
run: ## 📐 Starts the app in dev environment
	go run ./cmd/api -db-dsn=${BOOKWORM_DB_DSN}

# .PHONY: air 
# air: ## 📐 Starts the app in dev environment with hot reload 🔥
# 	air

# .PHONY: test
# test: ## 🧪 Runs the tests
# 	go test -v ./...

# .PHONY: testnocache
# testnocache: ## 🧪 Runs the tests caching turned off
# 	go test -v -count=1  ./...

# .PHONY: coverage
# coverage: ## 📊 Displays test coverage
# 	go test -v ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

.PHONY: build
build: ## 🏗️  Builds the app
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api
	
.PHONY: db
db: ## 🚀 Starts PostgreSQL Docker container or builds one if doesn't exist.
	@if [ $$(docker ps -aq -f name=bookworm) ]; then \
		if [ ! $$(docker ps -q -f name=bookworm) ]; then \
			docker start bookworm; \
		fi; \
	else \
		docker run -d \
			--name bookworm \
			-e POSTGRES_USER=bookworm \
			-e POSTGRES_PASSWORD=b00kworm \
			-e POSTGRES_DB=bookworm \
			-p 5432:5432 \
			postgres; \
	fi

.PHONY: psql
psql: ## 🗄️  Connects to DB shell as user bookworm
	docker exec -it bookworm psql --host=localhost --dbname=bookworm --username=bookworm

.PHONY: new
new: ## 🗄️  Adds a new set of database migrations. Ex. 'make new users'
	$(eval TABLENAME=$(filter-out $@,$(MAKECMDGOALS)))
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations $(TABLENAME)
%:
	@:

.PHONY: up
up: confirm ## 🗄️  Runs up migrations
	@echo 'Running up migrations...'
	migrate -path ./migrations -database $(BOOKWORM_DB_DSN) up

# ══════════════════════════════════════════
#               QUALITY CONTROL             
# ══════════════════════════════════════════

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: vendor ## ✅ Runs vet, staticcheck, lint and tests
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	golangci-lint run -D errcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor: ## ✅ Runs go mod tidy, verify and vendor
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ══════════════════════════════════════════
#                 PRODUCTION                
# ══════════════════════════════════════════

production_host = 'bookworm.onatim.com'# IP or domain name

.PHONY: connect
connect: ## 📡 Connects to production server
	ssh bookworm@${production_host}

.PHONY: deploy
deploy: ## 🌐 Deploys the api to production
	rsync -P ./bin/linux_amd64/api bookworm@${production_host}:~
	rsync -rP --delete ./migrations bookworm@${production_host}:~
	rsync -P ./remote/production/api.service bookworm@${production_host}:~
	rsync -P ./remote/production/Caddyfile bookworm@${production_host}:~
	ssh -t bookworm@${production_host} '\
	  migrate -path ~/migrations -database $$BOOKWORM_DB_DSN up \
	  && sudo mv ~/api.service /etc/systemd/system/ \
	  && sudo systemctl enable api \
	  && sudo systemctl restart api \
	  && sudo mv ~/Caddyfile /etc/caddy/ \
	  && sudo systemctl reload caddy \
    '

# ══════════════════════════════════════════
#                 HELPERS                   
# ══════════════════════════════════════════

