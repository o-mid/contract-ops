.PHONY: test test-api test-web up down

test: test-api test-web

test-api:
	cd api && go test ./...

test-web:
	npm run check

up:
	docker compose up --build -d

down:
	docker compose down
