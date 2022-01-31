.PHONY: build
build:
	cd remoterootsync; go build .

.PHONY: vet
vet:
	go vet ./...
	cd remoterootsync; go vet ./...

.PHONY: test
test:
	go test ./...
	cd remoterootsync; go test ./...

.PHONY: ci
ci: build test vet
