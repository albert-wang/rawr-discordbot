# syntax=docker/dockerfile:1

FROM golang:1.26 AS builder
WORKDIR /go/src/nvg-tan
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build

FROM ubuntu:latest AS rust
RUN apt update && apt -y install ca-certificates tzdata build-essential rustup && rustup default stable
WORKDIR /src/nvg-tan/cr_epi
COPY cr_epi/Cargo.toml cr_epi/Cargo.lock cr_epi/main.rs ./
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/src/nvg-tan/cr_epi/target \
    touch main.rs && cargo build --release && cp target/release/cr_epi /cr_epi

FROM ubuntu:latest
RUN apt update && apt -y install ca-certificates imagemagick tzdata

COPY --from=builder /go/src/nvg-tan/rawr-discordbot .
COPY --from=rust /cr_epi ./cr_episode

EXPOSE 14001

ENTRYPOINT [ "./rawr-discordbot" ]
