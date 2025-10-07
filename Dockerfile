FROM docker.io/library/golang:1.25.2-alpine3.21 AS build

WORKDIR /ask

RUN apk update && apk add --no-cache ca-certificates git gcc musl-dev sqlite-dev
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-linkmode external -extldflags '-static' -s -w" -o ask

# -----------------------------------------------------------------------------
FROM alpine:3.22
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Create necessary directories with proper permissions
RUN mkdir -p /app /home/appuser/.ask && \
    chown -R appuser:appgroup /app /home/appuser

# Copy the binary and startup script
COPY --from=build --chown=appuser:appgroup /ask/ask /app/ask
COPY --chown=appuser:appgroup entrypoint.sh /app/entrypoint.sh
COPY --chown=appuser:appgroup templates /app/templates

# Make the startup script executable
RUN chmod +x /app/entrypoint.sh

# Switch to non-root user
USER appuser
WORKDIR /app

# Set environment variables
ENV PATH="/app:${PATH}"
ENV HOME="/home/appuser"

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
