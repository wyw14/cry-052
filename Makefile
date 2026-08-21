.PHONY: test race vet build web-test web-build check run

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build ./...

web-test:
	cd web && npm test

web-build:
	cd web && npm run build

check: test race vet build web-test web-build

run:
	go run ./cmd/server
