# Stage 1: Frontend build
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY server/frontend/package*.json ./
RUN npm ci
COPY server/frontend/ ./
RUN npm run build

# Stage 2: Go build
FROM golang:1.24-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
# Copy the vanilla PWA files as the embedded web directory
COPY src/ ./web/
# Overlay with React frontend build output
COPY --from=frontend /app/frontend/dist/ ./web/
RUN CGO_ENABLED=1 go build -o familycall .

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=backend /app/familycall /usr/local/bin/
EXPOSE 443 8080 3478/udp
VOLUME ["/data"]
ENV PORT=""
ENV DISABLE_TURN=""
ENV FRONTEND_URI=""
ENTRYPOINT ["familycall"]
