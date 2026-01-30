# Build the frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/ui
COPY ui/package.json ui/pnpm-lock.yaml ./
COPY ui/patches ./patches
RUN npm install -g pnpm && pnpm install
COPY ui/ .
RUN pnpm build

# Build the backend
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
# Copy frontend build artifacts to backend static directory
COPY --from=frontend-builder /app/ui/dist ./static
RUN CGO_ENABLED=0 GOOS=linux go build -o steer .

# Final image
FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates git helm curl
COPY --from=backend-builder /app/backend/steer .
# Create directory for static files if needed, though binary should embed them or serve from relative path
# Assuming the backend serves static files from ./static relative to the binary
COPY --from=backend-builder /app/backend/static ./static

EXPOSE 8080
ENTRYPOINT ["./steer"]
