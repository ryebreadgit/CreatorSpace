# Stage 1: Build stage
FROM golang:1.20.4-alpine3.18 AS build

RUN apk update && \
    apk add --no-cache git

# Download the build dependencies
RUN apk add --no-cache git make build-base

WORKDIR /go/CreatorSpace

# Copy the local repository to the build context
COPY . .

# Change the working directory to the cmd folder

WORKDIR /go/CreatorSpace/cmd

# Enable CGO
ENV CGO_ENABLED=1

# Build the application
RUN go get && go build -o ../cs

# Stage 2: Final stage
FROM alpine:latest AS final

# Create the user inside the Docker image
ARG UID=1000
ARG GID=1000

RUN addgroup -g $GID -S csgroup && \
    adduser -u $UID -S csuser -G csgroup

RUN apk update && apk add --no-cache wget
RUN apk add --no-cache python3 py3-pip ffmpeg
RUN mkdir /CreatorSpace && chown csuser:csgroup /CreatorSpace

# Change to the new user in the Docker image
USER csuser

# Add pip binaries to PATH
ENV PATH="/home/csuser/.local/bin:${PATH}"

# Install python dependencies
RUN pip3 install --upgrade pip
RUN pip3 install yt-dlp

# Change the working directory /CreatorSpace
WORKDIR /CreatorSpace/

# Set the environment variable
ENV GIN_MODE=release

# Copy the binary from the build stage
COPY --from=build /go/CreatorSpace/cs ./cs

# Copy the templates, config, and static folders
COPY --from=build /go/CreatorSpace/templates ./templates
COPY --from=build /go/CreatorSpace/config ./config
COPY --from=build /go/CreatorSpace/static ./static

# Switch to the root user
USER root

# Add user permissions to the folders
RUN chown -R csuser:csgroup /CreatorSpace
RUN chmod -R 755 /CreatorSpace

# Switch back to the csuser
USER csuser

# Set the entrypoint
ENTRYPOINT ["/CreatorSpace/cs"]
