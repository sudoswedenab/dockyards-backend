# Build-stage
FROM docker.io/library/golang:1.21 as builder

WORKDIR /usr/src/app
# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
# .env values will be passed from ConfigMap
ENV CGO_ENABLED=0
RUN go build -v -o ./dist/dockyards-backend ./cmd/dockyards-backend

# Deploy-stage
FROM gcr.io/distroless/static-debian11:nonroot
COPY --from=builder /usr/src/app/dist/dockyards-backend /dockyards-backend
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 9000
ENTRYPOINT ["/dockyards-backend"]
