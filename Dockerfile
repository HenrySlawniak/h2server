FROM golang:1.12.7-alpine as builder

WORKDIR /go/src/github.com/HenrySlawniak/h2server
RUN apk add git

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/h2server

FROM scratch

COPY --from=builder /go/bin/h2server /bin/h2server

ENTRYPOINT ["/bin/h2server"]
