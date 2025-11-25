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

COPY go.mod go.sum ./
COPY ./migrations ./migrations
COPY --from=builder /srv/.dist/aoclb .

CMD ["./aoclb"]
