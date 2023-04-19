FROM golang:latest as build-env

ENV CGO_ENABLED=0

COPY . /app/
WORKDIR /app/

RUN mkdir /data && \
    go build -o backend -ldflags="-extldflags=-static -w" .

FROM gcr.io/distroless/static

COPY --from=build-env --chown=nonroot:nonroot \
    /app/backend \
    /app/

COPY --from=build-env --chown=nonroot:nonroot \
    /data \
    /data

USER nonroot:nonroot
WORKDIR /app/

ENTRYPOINT [ "/app/backend" ]