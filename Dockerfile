FROM golang:1.9 as builder

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.3.1/dep-linux-amd64 && chmod +x /usr/local/bin/dep

WORKDIR /go/src/github.com/saracen/navigator

COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o navigator .

FROM scratch
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=builder /go/src/github.com/saracen/navigator/navigator .

ENTRYPOINT ["./navigator"]