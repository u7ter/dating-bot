.PHONY: build run stop clean logs scale monitor

# Build and run with high-load configuration
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

# Monitor system
monitor:
	@echo "Opening monitoring dashboards..."
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Health Check: http://localhost/health"

# Performance testing
load-test:
	@echo "Running load test..."
	docker run --rm -i --network dating_bot_network \
		grafana/k6 run --vus 100 --duration 30s - < load-test.js

# Database operations
db-backup:
	docker-compose exec mysql mysqldump -u root -p$(MYSQL_ROOT_PASSWORD) dating_bot > backup_$(shell date +%Y%m%d_%H%M%S).sql

db-restore:
	@echo "Usage: make db-restore FILE=backup_file.sql"
	@if [ -z "$(FILE)" ]; then echo "Please specify FILE=backup_file.sql"; exit 1; fi
	docker-compose exec -T mysql mysql -u root -p$(MYSQL_ROOT_PASSWORD) dating_bot < $(FILE)

# Redis operations
redis-cli:
	docker-compose exec redis redis-cli

redis-monitor:
	docker-compose exec redis redis-cli monitor

redis-info:
	docker-compose exec redis redis-cli info

# Development mode with auto-reload
dev:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build

# Production deployment
prod:
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d --scale bot=5

# Security scan
security-scan:
	docker run --rm -v $(PWD):/app securecodewarrior/docker-security-scan /app

# Update dependencies
update-deps:
	docker-compose exec bot go mod tidy
	docker-compose exec bot go mod download

# Stress test specific components
stress-test-db:
	docker-compose exec mysql mysqlslap --user=root --password=$(MYSQL_ROOT_PASSWORD) \
		--host=localhost --concurrency=50 --iterations=100 --create-schema=dating_bot

stress-test-redis:
	docker-compose exec redis redis-cli eval "for i=1,10000 do redis.call('set', 'key'..i, 'value'..i) end" 0

# Check system resources
resources:
	@echo "=== Docker Stats ==="
	docker stats --no-stream
	@echo "=== Disk Usage ==="
	docker system df
	@echo "=== Network Usage ==="
	docker network ls

# Cleanup old data
cleanup-old-data:
	docker-compose exec redis redis-cli EVAL "return redis.call('del', unpack(redis.call('keys', 'rate_limit:*')))" 0
	docker-compose exec mysql mysql -u root -p$(MYSQL_ROOT_PASSWORD) -e "DELETE FROM dating_bot.likes WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);"

# Generate SSL certificates
ssl-certs:
	mkdir -p ssl
	openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
		-keyout ssl/nginx.key -out ssl/nginx.crt \
		-subj "/C=UA/ST=Kyiv/L=Kyiv/O=DatingBot/CN=localhost"