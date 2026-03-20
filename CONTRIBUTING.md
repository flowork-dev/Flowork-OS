# Contributing to Flowork OS

First off, thank you for considering contributing to Flowork OS!
We welcome developers to build powerful "Local Apps" and "Nodes" for our Hybrid Ecosystem. To maintain maximum performance (60fps UI) and absolute security, all contributors must strictly adhere to our development laws.

## 🏗️ The Dual-Engine Architecture
Before building an app, understand our core concept:
1. **The Web UI (Online)**: Pure HTML/JS handling the visual interface.
2. **The Local Engine (Offline)**: The Golang Engine executing heavy Python, Node.js, or C++ scripts natively on the user's hardware.
3. **The P2P Tunnel**: Communication happens via a secure WebSocket using `systemBridge.js`.

## 📜 The "Absolute Laws" of App Development

If you are submitting a new Local App to our registry, your app will be rejected if it violates any of the following rules:

### 1. English Only in Code
All variables, function names, and developer comments in your source code (Python, JS, Go) MUST be written in English.

### 2. Zero Logic in HTML
You are strictly forbidden from placing logic or event handlers inside HTML tags (e.g., `<button onclick="start()">`). Use the custom attribute `data-flowork-action` and let your `app.js` handle the event delegation.

### 3. Zero Hardcoded UI Text (Strict i18n)
Do not write UI text directly in your HTML. All text must be called dynamically using `data-i18n` attributes linked to your `i18n.json` dictionary. This ensures seamless OS-level auto-translation.

### 4. Zero External UI Scripts
Do not use external CDNs (like Bootstrap or Tailwind) in your UI. The UI must be built using pure Vanilla CSS and JavaScript to ensure offline compatibility.

### 5. Mobile UX Strictness
If your app supports Android/Mobile (`mobile.html`), you must apply:
- **Edge-to-Edge Layout**: No margin cards.
- **Dock Safe Area**: Include `padding-bottom: 85px;` to prevent OS overlap.
- **Zero Blur**: Do not use `backdrop-filter: blur()`. Use solid RGBA colors to save GPU performance.

### 6. Portable Sandbox Execution
Do not instruct users to open their terminal. List your dependencies in a `requirements.txt` or `package.json` file. The Engine will automatically download and isolate them into a local `/libs` folder.

## 📂 Standard App Anatomy
Your app folder (e.g., `apps/your-app-name/`) must contain these exact files:
- `manifest.json` (App identity and "is_local": true flag)
- `schema.json` (I/O standard contract)
- `script.py` / `script.js` (The native backend brain)
- `index.html` & `mobile.html` (The UI files)
- `app.js` (Frontend logic)
- `systemBridge.js` (P2P gateway)
- `i18n.json` (Language dictionary)

For a full step-by-step masterclass tutorial, visit our blog at [https://floworkos.com/blog](https://floworkos.com/blog).