FROM docker.io/library/golang:1.21 as builder
COPY . /src
WORKDIR /src
ENV CGO_ENABLED=0
RUN go build -v -o ./dockyards-backend ./cmd/dockyards-backend

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /src/dockyards-backend /usr/bin/dockyards-backend
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 9000
ENTRYPOINT ["/usr/bin/dockyards-backend"]
