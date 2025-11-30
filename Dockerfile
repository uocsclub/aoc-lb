FROM golang:trixie AS builder

WORKDIR /srv

COPY go.mod go.sum Makefile ./

RUN make tailwind-install

COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./tailwind.config.js .

RUN make build

FROM debian:trixie AS runner

WORKDIR /srv

RUN apt update && \
    apt install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
COPY ./migrations ./migrations
COPY --from=builder /srv/.dist/aoclb .
RUN touch fiber_storage.sqlite3 data.sqlite3

CMD ["./aoclb"]
