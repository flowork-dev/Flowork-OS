# Changelog

All notable changes to the Flowork OS Engine will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-03-12

### Added
- **Dual-Engine Architecture**: Initial release of the Golang-based offline engine that pairs with the Flowork Cloud UI.
- **P2P WebSocket Tunnel**: Secure, encrypted localhost communication between the frontend UI and the local hardware (`systemBridge.js`).
- **God Mode Integration**: Unrestricted cross-domain HTTP requests and web scraping capabilities via the Flowork Chrome Extension bridge.
- **Portable Sandbox Auto-Installer**: Automatic detection and installation of `requirements.txt` into a hidden `/libs` directory to prevent OS pollution.
- **Smart Kill-Switch**: Hardcoded security measure that safely shuts down the local engine if a version mismatch or server maintenance is detected.

### Fixed
- Stabilized JIT (Just-In-Time) execution for heavy native Python and C++ scripts.
- Resolved memory leak issues by enforcing strict garbage collection when the `stop` signal is received from the UI.