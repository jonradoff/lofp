# Stage 1: Build frontend
FROM node:22-alpine AS frontend
ARG VITE_GOOGLE_CLIENT_ID
ENV VITE_GOOGLE_CLIENT_ID=$VITE_GOOGLE_CLIENT_ID
ARG VITE_TURNSTILE_SITE_KEY
ENV VITE_TURNSTILE_SITE_KEY=$VITE_TURNSTILE_SITE_KEY
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend
WORKDIR /build
COPY engine/go.mod engine/go.sum ./
RUN go mod download
COPY engine/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /lofp ./cmd/lofp

# Stage 3: Production image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app

# Copy binary
COPY --from=backend /lofp .

# Copy config
COPY engine/config/prod.yaml config/prod.yaml

# Copy original scripts
COPY original/scripts/ scripts/

# Copy built frontend
COPY --from=frontend /build/dist static/

ENV LOFP_CONFIG=config/prod.yaml
ENV LOFP_STATIC_DIR=/app/static

EXPOSE 8080 4000 4001 4022
CMD ["./lofp"]
