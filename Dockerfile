FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o mock-server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/mock-server .
EXPOSE 8080
ENTRYPOINT ["./mock-server"]
