.PHONY: run build test test-integration docker-up docker-down start

migrate:
	@echo "🔨 Rodando as migrações do banco de dados..."
	go run internal/infra/database/migration/main.go

run:
	@echo "🚀 Rodando a aplicação localmente..."
	go run main.go

build:
	@echo "🔨 Construindo o binário localmente..."
	go build -o atom-ly

test:
	@echo "🧪 Rodando testes..."
	go test ./...

test-integration:
	@echo "🧪 Rodando testes de integração..."
	go test -tags=integration ./...

docker-up:
	@echo "🐳 Construindo a imagem Docker..."
	docker-compose up -d

docker-down:
	@echo "🛑 Parando o container..."
	docker-compose down -v

start: docker-up run
	@echo "🚀 Aplicação inicializada"
