FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/logpilot ./cmd/server

FROM alpine:3.20
RUN addgroup -S logpilot && adduser -S -G logpilot logpilot
WORKDIR /app
COPY --from=build /out/logpilot /usr/local/bin/logpilot
RUN mkdir -p /app/data && chown -R logpilot:logpilot /app
USER logpilot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/logpilot"]
