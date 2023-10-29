ARG GOLANG_VERSION=1.21
ARG ALPINE_VERSION=3.18
FROM golang:${GOLANG_VERSION}-alpine${ALPINE_VERSION}

WORKDIR /app
RUN apk add --no-cache --update git build-base iptables bash openresolv iproute2 iproute2-ss curl alpine-sdk
ENV GOPATH="/root/go"
ENV PATH="$PATH:$GOPATH/bin"
RUN go install github.com/mitranim/gow@latest
RUN go install -v golang.org/x/tools/gopls@latest
RUN go install -v golang.org/x/tools/cmd/goimports@latest
RUN go install -v github.com/rogpeppe/godef@latest
RUN go install -v github.com/stamblerre/gocode@latest

CMD ["gow run ./src"]