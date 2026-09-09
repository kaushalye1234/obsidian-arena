FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN adduser -D -H game
USER game
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
