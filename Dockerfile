FROM golang:1.8

RUN mkdir -p /go/src/tq
WORKDIR /go/src/tq

COPY . /go/src/tq
RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run"]
