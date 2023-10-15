build:
	go build -o mysql-backup main.go

run:
	go run main.go

lint:
	golangci-lint run

compose:
	docker-compose -f docker-compose.dev.yml watch

