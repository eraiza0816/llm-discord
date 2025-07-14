FROM golang:1.22

WORKDIR /go/src/app

# Install git and other tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Install delve for debugging
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.59.1

# Copy the current project to the container
COPY . /go/src/app

# Set up the entrypoint script
COPY .devcontainer/devcontainer-entrypoint.sh /usr/local/bin/devcontainer-entrypoint.sh
RUN chmod +x /usr/local/bin/devcontainer-entrypoint.sh
ENTRYPOINT ["devcontainer-entrypoint.sh"]
