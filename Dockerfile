FROM golang:1.25 as builder
WORKDIR src/nvg-tan
ADD . .
RUN go build

from ubuntu:latest as rust
RUN apt update && apt -y install ca-certificates imagemagick tzdata rustup build-essential && rustup default stable
WORKDIR /src/nvg-tan
ADD ./cr_epi ./cr_epi
WORKDIR cr_epi
RUN cargo build --release

FROM ubuntu:latest
RUN apt update && apt -y install ca-certificates imagemagick tzdata

COPY --from=builder /go/src/nvg-tan/rawr-discordbot .
COPY --from=rust /src/nvg-tan/cr_epi/target/release/cr_epi ./cr_episode

EXPOSE 14001

ENTRYPOINT [ "./rawr-discordbot" ]
