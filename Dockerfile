# syntax=docker/dockerfile:1

FROM golang:1.23-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd

FROM python:3.12-slim-bookworm
WORKDIR /app

RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir numpy

COPY --from=build /out/api /app/api
COPY python ./python
COPY data ./data

ENV HTTP_ADDR=:8080 \
    PYTHON_BIN=python3 \
    RUNNER_SCRIPT=python/runner.py \
    CORS_ORIGINS=*

EXPOSE 8080

USER nobody

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/health', timeout=2)"

ENTRYPOINT ["/app/api"]
