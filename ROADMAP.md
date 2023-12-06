# ROADMAP for CreatorSpace

## Current State

CreatorSpace is a comprehensive content archiving platform designed for YouTube creators. Here's a checklist of the features we've implemented and those that are still in progress:

---

## Known Issues

- [x] Theater mode video size is larger than theater mode container
- [ ] Cannot import existing library, only recommended for new setups for now
- [ ] Video download approval is required for non-admin users but there is no approval process
- [ ] Videos deleted from disk are not removed from library
- [x] Videos from 'Various Creators' are not moved to the correct creator if the creator is added later
  - [ ] Some metadata may not be moved correctly
- [ ] Mobile layout needs to be improved
- [ ] Needs refactoring to all hell

---

### YT-DLP Integration

- [x] Video Downloading
- [x] Metadata Downloading

### YouTube Integration

- [x] Video Downloading
- [x] Metadata Downloading
- [x] Channel Downloading
- [x] Playlist Downloading
- [ ] Subscription Downloading

### Task Management

- [x] Task Queuing
- [ ] Task Prioritization
- [ ] Task Progress Tracking
- [ ] Task Cancellation
- [ ] Task Resumption

### User Management

- [x] User Registration
- [x] User Login
- [x] JWT Authentication
- [x] User Settings
- [x] User Management
  - [x] Add new users
  - [x] Delete existing users
  - [x] Modify existing users permissions
  - [x] Add UI page for user management

### Video Management

- [x] Video Library
- [x] Video Search
- [ ] Import Videos
  - [ ] Import from CreatorSpace
  - [ ] Import from TubeArchivist
  - [ ] Import from YT-DLP folder
- [ ] Delete Videos
- [ ] Modify Video Metadata

### Playlist Management

- [x] Custom Playlists
- [x] Automatic Playlist Updates
- [ ] Playlist Management
- [ ] Add videos to playlists
- [ ] Remove videos from playlists

### Video Playback

- [x] SponsorBlock integration
- [x] VideoJS Player
- [ ] Theater Mode
- [ ] Picture-in-Picture Mode
- [x] Fullscreen Mode
- [x] Recommended Videos
- [x] Video Comments
  - [ ] View Comment User Profiles

### User Interface

- [x] Responsive Design
- [ ] Dark Mode
- [ ] Customizable Layout

### Backend

- [x] Go-based Server
- [x] JWT for Authentication
- [ ] Full test coverage
- [ ] Comprehensive documentation

---

## Future Plans

### Improved User Interface

- [ ] Implement a more intuitive and user-friendly design
- [ ] Improve navigation within the platform
- [ ] Enhance visual appeal and responsiveness

### Expanded Platform Support

- [ ] Add support for Twitch
  - [ ] Implement video downloading
  - [ ] Implement chat downloading
  - [ ] Fetch metadata via YT-DLP
  - [ ] Channel downloading task
- [x] Add support for Twitter (On hold until Twitter's API is fixed)
  - [x] Download tweets
  - [x] Download images
  - [x] Download videos
  - [x] Account downloading task
  - [ ] Hashtag downloading task
  - [ ] Tweet view page

### Expanded single video support

- [x] Add support for single video downloads
- [x] Add creator for unattributed videos
- [ ] Add better viewing experience for single videos
- [ ] Add easier way to download single videos in bulk

### Advanced Search Features

- [ ] Implement advanced search parameters
- [ ] Add filters for search results
- [ ] Improve search result sorting and presentation

### Library Sharing (?)

- [ ] Allow users to share and sync their libraries with other users via udp holepunch or p2p
- [ ] Allow users to download videos from other users' libraries if enabled
- [ ] Allow users to watch videos from other users' libraries from within their own library

### Mobile Application (?)

- [ ] Design and implement a mobile application for CreatorSpace
- [ ] Ensure feature parity with the web platform
- [ ] Flutter-based cross-platform application

### Localization

- [ ] Implement support for multiple languages
- [ ] Ensure all features and messages are properly localized
- [ ] Add language selection in user settings

Please note that this roadmap is subject to change based on user feedback and the evolving needs of our user base. We welcome contributions and suggestions from the community.
