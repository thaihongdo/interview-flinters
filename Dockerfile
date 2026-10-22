FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod .
COPY *.go .
RUN go build -o aggregator .

FROM alpine:3.20
COPY --from=build /app/aggregator /usr/local/bin/
ENTRYPOINT ["aggregator"]
