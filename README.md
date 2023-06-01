# CreatorSpace

![CreatorSpace Banner](./.github/img/CreatorSpaceBanner.png)

## Introduction

**CreatorSpace** is a comprehensive content archiving platform designed for YouTube creators.

## Features

- Archive First Approach
- Automatic Video & Metadata Downloading
- Automatic Metadata Updates
- Updates saved along-side previous versions
- Single Video and Playlist Archiving
- Sponsorblock Integration
- User and Library Management
- Robust API Server
- Enhanced Security
- Self-Hostable Backend
- Optimized Video Handling
- High-Performance Execution

## Tech Stack

- [Go](https://go.dev/)
- [Gin](https://gin-gonic.com/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [jwt](https://jwt.io/)

## Screenshots

<details>
<summary>Click to expand</summary>

![Subscriptions](./.github/img/subscriptions.png)

![Creator View](./.github/img/creator.png)

![Video View](./.github/img/video-playing.png)

![Comments and Recommendations](./.github/img/comments-recommendations.png)

![On-Disk Info](./.github/img/disk-creator.png)

</details>

## Getting Started

### Pre-built Binaries

You can download the pre-built binaries directly from the [releases](https://github.com/ryebreadgit/CreatorSpace/releases/latest) section. Available for Windows, macOS, and Linux.

### Build Instructions

If you prefer to build the application yourself:

1. Clone the repository
    ```shell
    git clone https://github.com/ryebreadgit/CreatorSpace.git
    ```
2. Change into the directory
    ```shell
    cd CreatorSpace
    ```
3. Change into the "cmd" directory
    ```shell
    cd cmd
    ```
4. Download dependencies
    ```shell
    go get
    ```
5. Build the application
    ```shell
    go build
    ```
6. Move the output to the root folder
    ```shell
    mv CreatorSpace ..
    ```

## Contributing

Contributions are always welcome! Please read the [contribution guidelines](CONTRIBUTING.md) first.

## License

This project is licensed under the [MIT License](LICENSE.md).