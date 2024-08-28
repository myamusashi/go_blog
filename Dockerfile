FROM golang:1.23.0-alpine3.19 as build
RUN mkdir -p /build/
WORKDIR /build

COPY markdown/ /build/
COPY static/ /build/
COPY templates/ /build/
COPY server.go /build/
COPY go.mod /build/
COPY go.sum /build/
RUN go mod download

RUN go build -o main

FROM alpine:3.19
COPY --from=build /build /build
WORKDIR /build

EXPOSE 8080
ENTRYPOINT [ "./main" ]
