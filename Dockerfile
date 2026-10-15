FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /ratewatch ./cmd/ratewatch

FROM alpine
RUN apk add --no-cache ca-certificates
COPY --from=build /ratewatch /ratewatch
CMD ["/ratewatch"]
