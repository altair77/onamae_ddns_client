FROM golang:1.16.7-buster as build

WORKDIR /go/src
COPY ./src /go/src

RUN go mod download
RUN go build -o /go/bin/onamae main.go

FROM gcr.io/distroless/base

WORKDIR /work

COPY --from=build /go/bin/onamae /usr/local/bin/onamae

CMD ["/usr/local/bin/onamae"]

