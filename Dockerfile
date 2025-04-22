# syntax=docker/dockerfile:1

FROM golang:1.24 AS build-stage

COPY src/ /src/
WORKDIR /src

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /hass-ecowitt-proxy

FROM gcr.io/distroless/base-debian11 AS release-stage

WORKDIR /

COPY --from=build-stage /hass-ecowitt-proxy /hass-ecowitt-proxy
COPY --from=build-stage /src/html/ /html/

EXPOSE 8181

USER nonroot:nonroot

CMD ["/hass-ecowitt-proxy", "serve"]

# Metadata
LABEL org.opencontainers.image.source="https://github.com/organicveggie/home-assistant-ecowitt-proxy"
LABEL org.opencontainers.image.description="HTTP to HTTPS proxy for using Ecowitt Weather Stations with Home Assistant"
LABEL org.opencontainers.image.licenses=GPL-3.0-only
LABEL org.opencontainers.image.title="Home Assistant Ecowitt Proxy"
LABEL org.opencontainers.image.url="https://github.com/organicveggie/home-assistant-ecowitt-proxy"