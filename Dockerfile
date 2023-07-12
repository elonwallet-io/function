FROM golang:latest as build-env

ENV CGO_ENABLED=0

COPY . /app/
WORKDIR /app/

RUN mkdir /data && \
    go build -o wallet-service -ldflags="-extldflags=-static -w" .

FROM gcr.io/distroless/static

COPY --from=build-env --chown=nonroot:nonroot \
    /app/wallet-service \
    /app/

COPY --from=build-env --chown=nonroot:nonroot \
    /data \
    /data

USER nonroot:nonroot
WORKDIR /app/

ENTRYPOINT [ "/app/wallet-service" ]