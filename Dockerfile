ARG VERSION="dev"

FROM golang:1.24.0 AS build
# allow this step access to build arg
ARG VERSION
# Set the working directory
WORKDIR /build

# Install dependencies
COPY go.mod go.sum ./

COPY . ./
# Build the server
RUN  CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o kafka-mcp-server cmd/kafka-mcp-server/main.go

# Make a stage to run the app
FROM gcr.io/distroless/base-debian12
# Set the working directory
WORKDIR /server
# Copy the binary from the build stage
COPY --from=build /build/kafka-mcp-server .
# Command to run the server
ENTRYPOINT ["./kafka-mcp-server", "stdio"]
