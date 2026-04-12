DOCKER_IMAGE := hastenr/chatapi
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "latest")

.PHONY: build build-all build-linux build-macos build-windows docker-build docker-run docker-push clean test

# Build for current platform
build:
	go build -ldflags="-s -w" -o bin/chatapi ./cmd/chatapi

# Build for all platforms
build-all: build-linux build-macos build-windows

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/chatapi-linux-amd64 ./cmd/chatapi
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/chatapi-linux-arm64 ./cmd/chatapi

# Build for macOS
build-macos:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/chatapi-macos-amd64 ./cmd/chatapi
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/chatapi-macos-arm64 ./cmd/chatapi

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/chatapi-windows-amd64.exe ./cmd/chatapi

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):latest

# Push image to Docker Hub
docker-push: docker-build
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Clean build artifacts
clean:
	rm -rf bin/ chatapi

# Run tests
test:
	go test ./...