# Stage 1: Go build (vanilla frontend is embedded from server/web/)
FROM golang:1.24-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
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
