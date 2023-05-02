
test:
	docker-compose up -d db
	docker-compose up -d mysql
	go test -v -cover ./...

bench:
	docker-compose up -d db
	docker-compose up -d mysql
	go test -bench=.
.PHONY:bench

sample:
	docker-compose up -d db
	docker-compose up -d mysql
	go run cmd/sample/main.go
.PHONY:sample

sample_ctx:
	docker-compose up -d db
	docker-compose up -d mysql
	go run cmd/sample/main.go
.PHONY:sample_ctx

fmt:
	go fmt ./...
.PHONY:fmt

lint: fmt
	staticcheck ./...
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: vet
	go build github.com/jonbodner/proteus/cmd/sample
.PHONY:build

install:
	go install honnef.co/go/tools/cmd/staticcheck@latest
.PHONY:install
