FROM ubuntu:latest

RUN apt update && apt -y install ca-certificates imagemagick

ADD rawr-discordbot .
ADD config.json .

EXPOSE 14001

ENTRYPOINT [ "./rawr-discordbot" ]