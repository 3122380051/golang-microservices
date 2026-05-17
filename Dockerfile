FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE=api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app ./cmd/${SERVICE}

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /bin/app /usr/local/bin/app
EXPOSE 8080
CMD ["app"]
