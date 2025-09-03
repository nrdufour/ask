FROM docker.io/library/golang:1.25.1-alpine3.21 AS build

WORKDIR /ask

RUN apk update && apk add --no-cache tini-static
COPY . .
RUN go build -ldflags="-s -w"

# -----------------------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot

COPY --from=build /ask/ask /app/ask
COPY --from=build --chown=nonroot:nonroot /sbin/tini-static /sbin/tini

ENTRYPOINT ["/sbin/tini", "--", "/app/ask"]
