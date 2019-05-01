FROM ubuntu:latest

RUN apt update && apt -y install ca-certificates

ADD rawr-discordbot .
ADD config.json .

EXPOSE 14001

ENTRYPOINT [ "./rawr-discordbot" ]