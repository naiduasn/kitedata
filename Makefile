.PHONY: build clean run docker-build docker-run test

# Default Go build flags
GOBUILD=go build
GOCLEAN=go clean
GOTEST=go test
GOGET=go get

# Binary name
BINARY_NAME=kitedata

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd

# Install the application to $GOPATH/bin
install:
	go install ./cmd

# Clean the binary
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Run the application
run:
	./$(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Build Docker image
docker-build:
	docker build -t kitedata:latest .

# Run in Docker container with interactive terminal
docker-run:
	docker run -it --rm \
		-v $(shell pwd)/config.yaml:/app/config.yaml \
		-v $(shell pwd)/historical_data:/app/historical_data \
		-v $(shell pwd)/parquet_data:/app/parquet_data \
		kitedata:latest

# Create config file from example if it doesn't exist
config:
	@if [ ! -f config.yaml ]; then \
		cp config.yaml.example config.yaml; \
		echo "config.yaml created from example. Please edit with your settings."; \
	else \
		echo "config.yaml already exists."; \
	fi