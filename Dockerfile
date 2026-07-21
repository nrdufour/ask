# -----------------------------------------------------------------------------
# OCI image labels (set via --build-arg; defaults apply when omitted)
# IMAGE_REVISION and IMAGE_CREATED should be passed at build time
ARG IMAGE_SOURCE=https://forge.internal/nemo/airport-swiss-knife
ARG IMAGE_REVISION=unknown
ARG IMAGE_CREATED=unknown
ARG IMAGE_VERSION=0.1.0
ARG IMAGE_TITLE=airport-swiss-knife
ARG IMAGE_DESCRIPTION="A Go CLI tool that downloads airport data from ourairports.com and makes it searchable via a web interface"
ARG IMAGE_AUTHORS=nemo
ARG IMAGE_LICENSES=Apache-2.0
ARG IMAGE_VENDOR=ptinem
ARG IMAGE_URL=https://forge.internal/nemo/airport-swiss-knife
ARG IMAGE_DOCUMENTATION=https://forge.internal/nemo/airport-swiss-knife
# -----------------------------------------------------------------------------

FROM docker.io/library/golang:1.25.5-alpine3.21 AS build

WORKDIR /ask

RUN apk update && apk add --no-cache ca-certificates git gcc musl-dev sqlite-dev
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-linkmode external -extldflags '-static' -s -w" -o ask

# -----------------------------------------------------------------------------
FROM alpine:3.24

# Re-declare ARGs so they're available in this stage
ARG IMAGE_SOURCE
ARG IMAGE_REVISION
ARG IMAGE_CREATED
ARG IMAGE_VERSION
ARG IMAGE_TITLE
ARG IMAGE_DESCRIPTION
ARG IMAGE_AUTHORS
ARG IMAGE_LICENSES
ARG IMAGE_VENDOR
ARG IMAGE_URL
ARG IMAGE_DOCUMENTATION

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
COPY --chown=appuser:appgroup static /app/static

# Make the startup script executable
RUN chmod +x /app/entrypoint.sh

# OCI annotations (see https://specs.opencontainers.org/image-spec/annotations/)
LABEL org.opencontainers.image.source=$IMAGE_SOURCE
LABEL org.opencontainers.image.revision=$IMAGE_REVISION
LABEL org.opencontainers.image.created=$IMAGE_CREATED
LABEL org.opencontainers.image.version=$IMAGE_VERSION
LABEL org.opencontainers.image.title=$IMAGE_TITLE
LABEL org.opencontainers.image.description=$IMAGE_DESCRIPTION
LABEL org.opencontainers.image.authors=$IMAGE_AUTHORS
LABEL org.opencontainers.image.licenses=$IMAGE_LICENSES
LABEL org.opencontainers.image.vendor=$IMAGE_VENDOR
LABEL org.opencontainers.image.url=$IMAGE_URL
LABEL org.opencontainers.image.documentation=$IMAGE_DOCUMENTATION

# Switch to non-root user
USER appuser
WORKDIR /app

# Set environment variables
ENV PATH="/app:${PATH}"
ENV HOME="/home/appuser"

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
