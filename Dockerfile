# Stage 1: Build stage
FROM golang:1.20.4-alpine3.18 AS build

RUN apk update && \
    apk add --no-cache git

WORKDIR /go/CreatorSpace

# Copy the local repository to the build context
COPY . .

# Change the working directory to the cmd folder

WORKDIR /go/CreatorSpace/cmd

# Build the application
RUN go get && go build -o ../CreatorSpace

# Stage 2: Final stage
FROM alpine:latest

RUN apk update && apk add --no-cache wget
RUN apk add --no-cache python3 py3-pip ffmpeg
RUN pip3 install --upgrade pip
RUN pip3 install yt-dlp

WORKDIR /CreatorSpace

# Copy the binary from the build stage
COPY --from=build /go/CreatorSpace/CreatorSpace .

# Copy the templates, config, and static folders
COPY --from=build /go/CreatorSpace/templates ./templates
COPY --from=build /go/CreatorSpace/config ./config
COPY --from=build /go/CreatorSpace/static ./static

# Set the entrypoint
ENTRYPOINT ["./CreatorSpace"]
