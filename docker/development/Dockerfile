FROM golang:1.27-alpine


# Install development dependencies and Air for hot-reloading
RUN apk add --no-cache git make curl \
    && go install github.com/air-verse/air@latest

# 1. Accept PORT as a build argument with a default fallback
ARG PORT=8080

# 2. Export it as an ENV variable so the Go binary can read os.Getenv("PORT") at runtime
ENV PORT=${PORT}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE ${PORT}

# Fallback to standard go run if Air configuration (.air.toml) is missing
CMD ["air", "--build.cmd", "go build -o ./tmp/main ./cmd/server", "--build.bin", "./tmp/main"]