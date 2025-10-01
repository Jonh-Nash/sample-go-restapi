FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -o /out/app ./cmd/api-server

FROM gcr.io/distroless/base-debian12:nonroot
USER nonroot:nonroot
WORKDIR /app
COPY --from=build /out/app /app/api-server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/api-server"]
