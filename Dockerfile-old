# Stage 1: Build the Go application
FROM golang:1.24.3-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
ENV CGO_ENABLED=0
RUN go build -o /pa11y-go-server cmd/server/main.go

# Stage 2: Create the final image
FROM timbru31/node-chrome:20-slim

RUN apt-get update && apt-get install -y libglib2.0-dev
# Install pa11y globally
RUN npm install -g pa11y

# Ensure pa11y/puppeteer can find Chrome
ENV CHROME_BIN=/usr/bin/google-chrome \
    CHROME_PATH=/usr/bin/google-chrome \
    PUPPETEER_EXECUTABLE_PATH=/usr/bin/google-chrome

# Copy the built binary from the builder stage
COPY --from=builder /pa11y-go-server /pa11y-go-server

# Create a non-root user to run the application
RUN groupadd -r appgroup && useradd -r -g appgroup -ms /bin/bash appuser
RUN chown -R appuser:appgroup /pa11y-go-server
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Set the command to run the application
CMD ["/pa11y-go-server"]
