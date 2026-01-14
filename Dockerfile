FROM golang:1.25-alpine

# Install dependencies required for fetching and building
RUN apk add --no-cache git make

WORKDIR /app

# Copy source code
COPY . .

# Build the loader
RUN go build -o plugin-loader main.go

# Run the loader
CMD ["./plugin-loader"]