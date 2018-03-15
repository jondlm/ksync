FROM golang:1.9-alpine as builder

ARG VERSION=development
WORKDIR /go/src/lazywatch

COPY . .

RUN go build -ldflags "-X main.version=${VERSION}"

FROM alpine:3.7

COPY --from=builder /go/src/lazywatch/lazywatch /usr/local/bin/lazywatch

CMD ["lazywatch", "--version"]
