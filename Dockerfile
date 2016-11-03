FROM golang:latest

VOLUME ["/go/src/github.com/cj123/random.gif/gifs"]

ADD . /go/src/github.com/cj123/random.gif

WORKDIR /go/src/github.com/cj123/random.gif

RUN go get .
RUN go build .

EXPOSE 8000

ENTRYPOINT /go/src/github.com/cj123/random.gif/random.gif
