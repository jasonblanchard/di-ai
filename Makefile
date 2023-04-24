migrate-up:
	migrate -database postgres://postgres:sekret@0.0.0.0:5432/postgres?sslmode=disable -path db/migrations up

migrate-down:
	migrate -database postgres://postgres:sekret@0.0.0.0:5432/postgres?sslmode=disable -path db/migrations down