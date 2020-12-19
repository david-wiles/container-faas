FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR $GOPATH/src/paas-server
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/paas-server ./cmd/paas/main.go

RUN chmod +x /go/bin/paas-server

FROM scratch

COPY --from=builder /go/bin/paas-server /go/bin/paas-server

ENTRYPOINT ["/go/bin/paas-server"]
