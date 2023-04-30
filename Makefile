psql-up:
	docker run --name greenlight-postgres -p 5432:5432 -e POSTGRES_PASSWORD=password -d postgres:alpine

psql-down:
	docker rm -f greenlight-postgres

psql-start:
	docker start greenlight-postgres

psql-stop:
	docker stop greenlight-postgres
psql:
	docker exec -it greenlight-postgres psql -U greenlight

psql-setup:
	docker exec -it greenlight-postgres psql -U postgres -c "CREATE DATABASE greenlight;"
	docker exec -it greenlight-postgres psql -U postgres -d greenlight -c "CREATE ROLE greenlight WITH LOGIN PASSWORD 'password';"
	docker exec -it greenlight-postgres psql -U postgres -d greenlight -c "CREATE EXTENSION IF NOT EXISTS citext;"
