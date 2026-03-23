# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# --- Runtime stage ---
FROM golang:1.25-alpine
WORKDIR /app
COPY --from=build /server .
EXPOSE 8080
ENTRYPOINT ["./server"]
