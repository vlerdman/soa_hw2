FROM golang:1.20.3-alpine
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go mod download
EXPOSE 9000
ENTRYPOINT ["go", "run", "cmd/server/main.go"]
