# build app
FROM golang:1.20-alpine3.18 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -ldflags="-s -w" -trimpath -o bin/privtracker

# build runner
FROM alpine:latest

ENV HOME="/config" \
    XDG_CONFIG_HOME="/config" \
    XDG_DATA_HOME="/config"

WORKDIR /app

VOLUME [ "/config" ]

COPY --from=builder /src/bin/privtracker /usr/local/bin/

EXPOSE 1337

ENTRYPOINT [ "/usr/local/bin/privtracker" ]