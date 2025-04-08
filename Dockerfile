# Use an official Go image as the build stage.
FROM golang:1.20-alpine AS builder

# Install necessary packages: yt-dlp and ffmpeg.
RUN apk add --no-cache yt-dlp ffmpeg

# Set the working directory.
WORKDIR /app

# Copy the Go module files and download dependencies.
# Copy the Go module files and download dependencies.
COPY go.mod go.sum ./
RUN go mod download


# Copy the rest of your application code.
COPY . .

# Build the application. The output binary is named "app".
RUN go build -o app .

# Use a minimal runtime image.
FROM alpine:latest

# Install any necessary runtime dependencies, if any.
# For this case, we assume that yt-dlp and ffmpeg are no longer needed at runtime;
# however, if they are needed (for example, if they are called during request handling),
# you should install them here as well:
RUN apk add --no-cache yt-dlp ffmpeg

WORKDIR /app
# Copy the binary from the builder stage.
COPY --from=builder /app/app .

# Expose the port defined by Railway. Railway sets the PORT environment variable.
EXPOSE 8080

# Start the application. It uses the PORT environment variable (defaulting to 8080 in your code).
CMD ["./app"]
