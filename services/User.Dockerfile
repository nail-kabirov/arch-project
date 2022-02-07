# Builder image
FROM golang:1.17.2-alpine3.14 AS build

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/user ./cmd/user/

# Final image from scratch
FROM scratch
COPY --from=build /app/bin/user /bin/user

EXPOSE 8000
ENTRYPOINT ["/bin/user"]
