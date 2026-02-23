BINARY=krangka

cfg:
	@cp configs/files/example.yaml configs/files/default.yaml

# command to run http server
# example: make http
http:
	@go run main.go http || true

# command to run database migration up
# example: make migrate-up repo=mariadb max=10
migrate-up:
	@go run main.go migrate up $(repo) $(max)

# command to run database migration down
# example: make migrate-down repo=mariadb max=10
migrate-down:
	@go run main.go migrate down $(repo) $(max)

# command to run database migration new
# argument: repo= repository name, name= migration file name
# example: make migrate-new repo=user name=add_new_table
migrate-new:
	@if [ -z "$(repo)" ] || [ -z "$(name)" ]; then \
		echo "Error: requires at least 2 arg(s), only received 0"; \
		echo "Usage:"; \
		echo "  application migrate new [repository] [migration_name] [flags]"; \
		echo ""; \
		echo "Flags:"; \
		echo "  -h, --help   help for new"; \
		echo ""; \
		echo "Global Flags:"; \
		echo "      --config string   config file (default is default.yaml)"; \
		exit 2; \
	fi; \
	go run main.go migrate new $(repo) $(name)

# command to run subscriber
# example: make subscriber
subscriber:
	@go run main.go subscriber

# command to run background worker
# argument: name= worker name
# example: make worker name=cleaning-todo
worker:
	@go run main.go worker $(name)

# command to generate swagger documentation
# example: make swag
swag:
	# install swag if not installed
	@if ! command -v swag &> /dev/null; then \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init --output internal/adapter/inbound/http/docs --parseDependency
	@sed -i.bak 's|BasePath:         "[^"]*"|BasePath:         ""|g' internal/adapter/inbound/http/docs/docs.go
	@rm -f internal/adapter/inbound/http/docs/docs.go.bak

# command to install cli to local bin
# example: make krangka-install
.PHONY: krangka-install
krangka-install:
	@echo "Installing krangka CLI..."
	@cd cli/krangka && go install
	@echo "krangka CLI installed successfully to $$(go env GOPATH)/bin/krangka"

# command to build main application
# example: make build
.PHONY: build
build:
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY} -mod=vendor -a -installsuffix cgo -ldflags '-w'


test:
	go test -v -cover -count=1 -failfast ./... -coverprofile="coverage.out"

dependency:
	@echo "> Installing the server dependencies ..."
	@go mod vendor

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

# Docker Compose commands
# The project name is defined in deployment/development_main.yaml using the 'name' field

# Start all services
# example: make docker-up
docker-up:
	@docker compose -f deployment/development_main.yaml up -d

# Stop all services
# example: make docker-down
docker-down:
	@docker compose -f deployment/development_main.yaml down

# Stop and remove all containers, networks, and volumes
# example: make docker-clean
docker-clean:
	@docker compose -f deployment/development_main.yaml down -v --remove-orphans

# Show status of services
# example: make docker-status
docker-status:
	@docker compose -f deployment/development_main.yaml ps

# View logs
# example: make docker-logs
docker-logs:
	@docker compose -f deployment/development_main.yaml logs -f

# Docker Compose commands for Kafka
# The project name is defined in deployment/development_kafka.yaml using the 'name' field
docker-up-kafka:
	@docker compose -f deployment/development_kafka.yaml up -d

# Stop all kafka services
# example: make docker-down-kafka
docker-down-kafka:
	@docker compose -f deployment/development_kafka.yaml down
	
# Stop and remove all kafka containers, networks, and volumes
# example: make docker-clean-kafka
docker-clean-kafka:
	@docker compose -f deployment/development_kafka.yaml down -v --remove-orphans
	
# Show status of services
# example: make docker-status-kafka
docker-status-kafka:
	@docker compose -f deployment/development_kafka.yaml ps
	
# View logs
# example: make docker-logs-kafka
docker-logs-kafka:
	@docker compose -f deployment/development_kafka.yaml logs -f
	

