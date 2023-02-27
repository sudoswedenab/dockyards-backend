# Build-stage
FROM golang:1.19

WORKDIR /usr/src/app
# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
# .env values will be passed from ConfigMap
#COPY .env .
RUN go build -v -o ./dist/app ./cmd/main.go

# Deploy-stage
FROM debian:bullseye-slim

WORKDIR /root/
COPY --from=0 /usr/src/app/dist/ ./
#COPY --from=0 /usr/src/app/.env .env
EXPOSE 9000
CMD ["./app"]