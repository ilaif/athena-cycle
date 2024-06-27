# Go Syncer Dockerfile
FROM golang:1.21-alpine

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go build -o /go/bin/app ./cmd/syncer

CMD [ "/go/bin/app" ]
