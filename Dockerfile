# syntax=docker/dockerfile:1

FROM golang:1.19 AS build-stage

WORKDIR #Directory

COPY go.mod go.sum ./

RUN go mod download

COPY #all my code

RUN CGO_ENABLED=0 GOOS=linux go build -o sla /recall

FROM recall

WORKDIR /

COPY --from=build-stage /recall /recall

EXPOSE 8080

USER nonroot:nonroot


CMD ["/recall"]

