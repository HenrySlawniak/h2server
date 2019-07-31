FROM golang:1.12.7-alpine as builder

WORKDIR /go/src/github.com/HenrySlawniak/h2server
RUN apk add git

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/h2server

EXPOSE 80/tcp
EXPOSE 443/tcp

ENTRYPOINT ["/bin/h2server"]
