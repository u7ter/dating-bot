.PHONY: build run stop clean logs scale

# Build and run the application
build:
	docker-compose build

run:
	docker-compose up -d

# Scale bot instances
scale:
	docker-compose up -d --scale bot=3

# Stop the application
stop:
	docker-compose down

# Clean up containers and volumes
clean:
	docker-compose down -v
	docker system prune -f

# View logs
logs:
	docker-compose logs -f

# View bot logs only
bot-logs:
	docker-compose logs -f bot

# View database logs only
db-logs:
	docker-compose logs -f mysql

# Restart the bot
restart-bot:
	docker-compose restart bot

# Access MySQL shell
mysql-shell:
	docker-compose exec mysql mysql -u root -p dating_bot

# Access Redis CLI
redis-cli:
	docker-compose exec redis redis-cli

# Check system resources
resources:
	@echo "=== Docker Stats ==="
	docker stats --no-stream
	@echo "=== Disk Usage ==="
	docker system df

# Development mode
dev:
	docker-compose up --build

# Production mode
prod:
	docker-compose -f docker-compose.yml up -d --scale bot=5

# Run tests
test:
	go test -v ./...

# Load test
load-test:
	@echo "Running load test..."
	docker run --rm --network dating_bot_network \
		grafana/k6 run --vus 10 --duration 30s - < /dev/stdin <<< 'import http from "k6/http"; export default function() { http.get("http://nginx/health"); }'