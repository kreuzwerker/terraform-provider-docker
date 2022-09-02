FROM golang:1.18-alpine as builder
ARG MAIN_FILE_PATH

RUN echo "appuser:x:65534:65534:Appuser:/:" > /etc/passwd
WORKDIR /build
COPY $MAIN_FILE_PATH main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o server main.go

FROM alpine

RUN apk add --no-cache curl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY configs.json /configs.json
COPY secrets.json /secrets.json
COPY --from=builder /build/server /server

USER appuser
ENTRYPOINT ["/server"]
EXPOSE 8080
