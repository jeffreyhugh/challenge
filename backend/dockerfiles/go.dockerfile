FROM golang:alpine

ARG SUBMISSION_ID

RUN adduser -D challenge

WORKDIR /usr/src/app

COPY ./submissions/$SUBMISSION_ID.go ./main.go

RUN chown -R challenge:challenge ./
USER challenge

RUN go build main.go > ./compile.log
ENTRYPOINT ./main