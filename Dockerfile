# syntax=docker/dockerfile:1

FROM golang:1.19alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o / handler

EXPOSE 8080

CMD [ "/handler" ]