FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build \
    -ldflags "-X github.com/pcfreak30/agents-fileshare-mcp-server/internal/build.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /mcp-fileshare ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /mcp-fileshare /mcp-fileshare
EXPOSE 8080
ENTRYPOINT ["/mcp-fileshare"]
