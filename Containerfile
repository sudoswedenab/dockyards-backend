# Build-stage
FROM docker.io/library/golang:1.21 as builder
COPY . /src
WORKDIR /src
ENV CGO_ENABLED=0
RUN go build -v -o ./dockyards-backend ./cmd/dockyards-backend

# Deploy-stage
FROM gcr.io/distroless/static-debian11:nonroot
COPY --from=builder /src/dockyards-backend /dockyards-backend
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 9000
ENTRYPOINT ["/dockyards-backend"]
