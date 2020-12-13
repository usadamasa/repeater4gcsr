FROM golang:1.13 AS build-env

WORKDIR /app
COPY . .

ENV CGO_ENABLED 0

ENV GOOS linux
ENV GOARCH amd64
RUN go build -o main ./main.go

FROM gcr.io/distroless/base

WORKDIR /app
COPY --from=build-env /app/main /app/main

ENTRYPOINT ["./main"]
