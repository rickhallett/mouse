FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /out/mouse ./cmd/mouse

FROM alpine:3.20
RUN adduser -D -u 10001 mouse
USER mouse
WORKDIR /app
COPY --from=build /out/mouse /app/mouse
COPY config /app/config
COPY runtime /app/runtime
EXPOSE 8080
ENTRYPOINT ["/app/mouse"]
CMD ["-config", "/app/config/mouse.yaml", "-addr", ":8080"]
