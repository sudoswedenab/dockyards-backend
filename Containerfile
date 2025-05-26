FROM docker.io/library/golang:1.24.2 as builder
COPY . /src
WORKDIR /src
ENV CGO_ENABLED=0
RUN go build -o dockyards-backend -ldflags="-s -w"

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /src/dockyards-backend /usr/bin/dockyards-backend
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chown=65532:65534 /src/cue.mod /home/nonroot/cue.mod
EXPOSE 9000
ENTRYPOINT ["/usr/bin/dockyards-backend"]
