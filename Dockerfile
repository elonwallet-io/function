FROM golang:latest as build-env

ENV CGO_ENABLED=0

COPY . /app/
WORKDIR /app/

RUN go build -o backend -ldflags="-extldflags=-static -w" .

FROM gcr.io/distroless/static

COPY --from=build-env --chown=nonroot:nonroot \
    /app/backend \
    /app/

USER nonroot:nonroot
WORKDIR /app/

ENTRYPOINT [ "/app/backend" ]