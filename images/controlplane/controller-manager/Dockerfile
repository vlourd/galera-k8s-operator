# Build stage
FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o controller-manager ./cmd/controller-manager/

# Runtime stage
FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/controller-manager /
CMD ["/controller-manager"]
