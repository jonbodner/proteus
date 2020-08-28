
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

fmt:
	go fmt ./...
.PHONY:fmt

lint: fmt
	golint ./...
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: vet
	go build github.com/jonbodner/proteus/cmd/sample
.PHONY:build
