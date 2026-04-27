FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /spec-torture ./cmd/spec-torture

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /spec-torture /usr/local/bin/spec-torture
COPY specs/ /specs/

ENTRYPOINT ["spec-torture"]
