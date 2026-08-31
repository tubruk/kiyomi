.PHONY: check-providers
check-providers:
	./scripts/check-providers.sh

.PHONY: build
build:
	CGO_ENABLED=0 go build ./...

.PHONY: test
test:
	CGO_ENABLED=0 go test ./...

.PHONY: vet
vet:
	CGO_ENABLED=0 go vet ./...
