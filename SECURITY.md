# Security Policy

Security and privacy are the core foundations of Flowork OS. Because our engine runs as a Local Server (`localhost`) and executes native scripts (Python, Node.js, C++) directly on your hardware, we enforce strict security measures to protect your system.

## Supported Versions

We only provide security updates and support for the **latest release** of the Flowork OS Engine.

| Version | Supported          |
| ------- | ------------------ |
| v1.0.1  | :white_check_mark: |
| < v1.0.1| :x: (Kill-Switch)  |

*Note: For maximum security, our Smart Kill-Switch will automatically terminate the engine if an outdated version attempts to connect to the main ecosystem.*

## Zero Data Collection Policy
Flowork OS utilizes your own PC as a Private Server. We strictly **do not** collect, monitor, or store any user files, execution history, or processed data on our cloud servers. Your data never leaves your local machine.

## Reporting a Vulnerability

We take all security vulnerabilities seriously. **Please DO NOT report security vulnerabilities through public GitHub issues or discussions.**

If you discover a security vulnerability within Flowork OS, please send an email directly to our security team at:
📧 **security@floworkos.com**

Please include the following information in your report:
- Type of vulnerability (e.g., XSS, buffer overflow, unauthorized local access).
- Step-by-step instructions to reproduce the vulnerability.
- Proof-of-concept or exploit code (if available).
- The potential impact of the vulnerability.

We will acknowledge receipt of your vulnerability report within 48 hours and strive to send you regular updates about our progress. If the vulnerability is accepted, we will issue a patch in the next immediate release.