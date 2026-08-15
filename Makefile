.PHONY: check-providers
check-providers:
	./scripts/check-providers.sh

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...
