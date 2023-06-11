FROM golang:1.20.3-alpine
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go mod download
ENTRYPOINT ["go", "run", "cmd/client/main.go"]