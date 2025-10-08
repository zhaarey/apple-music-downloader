FROM ubuntu:24.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install required packages
RUN apt-get update && apt-get install -y \
    gpac \
    golang \
    ffmpeg \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create and set working directory
WORKDIR /app


# Copy the application files
COPY . .

# Download Go dependencies
RUN go mod download

# Create necessary directories
RUN mkdir -p /app/output_alac \
    /app/output_aac \
    /app/output_atmos \
    /app/converted_output_alac \
    /app/converted_output_aac \
    /app/converted_output_atmos

    # Set entrypoint
ENTRYPOINT ["go", "run", "main.go"]