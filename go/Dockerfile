# Start from the official Golang image to build the binary.
FROM golang:1.20.3 as builder

# Set the current working directory inside the container.
WORKDIR /app

# Copy go mod and sum files, then download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container.
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Use a small Alpine image to reduce the final image size.
FROM alpine:latest

# Set the current working directory.
WORKDIR /root/

# Copy the binary from the builder stage.
COPY --from=builder /app/main .

# Expose the port the app runs on.
EXPOSE 8080

# Command to run the binary.
CMD ["./main"]
