FROM golang:latest as builder
WORKDIR src/nvg-tan
ADD . .
RUN go build

FROM ubuntu:latest
RUN apt update && apt -y install ca-certificates imagemagick tzdata

COPY --from=builder /go/src/nvg-tan/rawr-discordbot .

EXPOSE 14001

ENTRYPOINT [ "./rawr-discordbot" ]
