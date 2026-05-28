ARG GO_VERSION=1.26-alpine

FROM golang:${GO_VERSION} AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/prasankit-api ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /out/prasankit-api /app/prasankit-api

EXPOSE 8080

CMD ["/app/prasankit-api"]
