# ============================================================
# Stage 1: Builder
# ============================================================

FROM golang:1.26.2-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/recall-server ./cmd/server


# ============================================================
# Stage 2: Runtime
# ============================================================

FROM alpine:3.20

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/recall-server /bin/recall-server 
COPY --from=builder /src/internal/storage/schemas /migrations 

RUN mkdir -p /data/storage

ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV STORAGE_PATH=/data/storage

EXPOSE 8080

ENTRYPOINT ["/bin/recall-server"]
