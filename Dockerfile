FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR $GOPATH/src/vlab/faas-server
COPY . .

RUN go mod download
RUN go mod verify

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/faas-server ./cmd/vlab-faas/main.go

RUN chmod +x /go/bin/faas-server

FROM scratch

COPY --from=builder /go/bin/faas-server /go/bin/faas-server

ENTRYPOINT ["/go/bin/faas-server"]
