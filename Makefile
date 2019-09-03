
test:
	sudo docker-compose up -d db
	go test -v -cover ./...

bench:
	sudo docker-compose up -d db
	go test -bench=.
.PHONY:bench

sample:
	sudo docker-compose up -d db
	go run cmd/sample/main.go
.PHONY:sample

sample_ctx:
	sudo docker-compose up -d db
	go run cmd/sample/main.go
.PHONY:sample_ctx
