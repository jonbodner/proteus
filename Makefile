
test:
	docker-compose up -d db
	go test -v -cover ./...

bench:
	docker-compose up -d db
	go test -bench=.
.PHONY:bench

sample:
	docker-compose up -d db
	go run cmd/sample/main.go
.PHONY:sample

sample_ctx:
	docker-compose up -d db
	go run cmd/sample/main.go
.PHONY:sample_ctx
