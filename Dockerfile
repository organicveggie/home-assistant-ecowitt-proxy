# syntax=docker/dockerfile:1

FROM golang:1.21 AS build-stage

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

CMD ["/hass-ecowitt-proxy"]
