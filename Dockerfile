# Stage 1: Build stage
FROM golang:1.20.4-alpine3.18 AS build

ENV APPVERSION=0.3.1

RUN apk update && \
    apk add --no-cache git && \
    apk add --no-cache git make build-base

WORKDIR /go/CreatorSpace

# Copy the local repository to the build context
COPY . .

# Change the working directory to the cmd folder

WORKDIR /go/CreatorSpace/cmd

# Try to get the commit hash and date, default to "Unknown"
ARG COMMIT_HASH="Unknown"
ARG BUILD_DATE="Unknown"
ARG GO_VERSION="Unknown"

# Enable CGO
ENV CGO_ENABLED=1

# Build the application
RUN export COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo $COMMIT_HASH) && \
    export BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo $BUILD_DATE) && \
    export GO_VERSION=$(go version 2>/dev/null || echo $GO_VERSION) && \
    go get && \
    go build -ldflags "-X github.com/ryebreadgit/CreatorSpace/internal/api.GitCommit=$COMMIT_HASH -X github.com/ryebreadgit/CreatorSpace/internal/api.BuildDate=$BUILD_DATE -X github.com/ryebreadgit/CreatorSpace/internal/api.AppVersion=$APPVERSION" -o /go/CreatorSpace/cs
 
# Stage 2: Final stage
FROM alpine:latest AS final

# Create the user inside the Docker image
ARG UID=1000
ARG GID=1000

RUN addgroup -g $GID -S csgroup && \
    adduser -u $UID -S csuser -G csgroup && \
    apk update && \
    apk add --no-cache wget && \
    apk add --no-cache python3 py3-pip ffmpeg && \
    mkdir /CreatorSpace && chown csuser:csgroup /CreatorSpace

# Change to the new user in the Docker image
USER csuser

# Add pip binaries to PATH
ENV PATH="/home/csuser/.local/bin:${PATH}"

# Install python dependencies
RUN pip3 install --upgrade pip && \
    pip3 install --force-reinstall https://github.com/yt-dlp/yt-dlp/archive/master.tar.gz

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
RUN chown -R csuser:csgroup /CreatorSpace && \
    chmod -R 755 /CreatorSpace

# Switch back to the csuser
USER csuser

# Set the entrypoint
ENTRYPOINT ["/CreatorSpace/cs"]
