# syntax=docker/dockerfile:1

FROM golang:1.19-alpine

WORKDIR /app

COPY *.go ./

RUN go build -o /handler

EXPOSE 8080

CMD [ "/handler" ]
