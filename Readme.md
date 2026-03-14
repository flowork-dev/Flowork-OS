# 🚀 Flowork OS

<div align="center">
  <img src="cover.png" alt="Flowork OS Banner" width="100%">
</div>


> **A Hybrid Operating System that Breaks Browser Limits & Unleashes Your Hardware's Potential.**

Ever felt like your favorite *web app* is getting heavier, eating up your RAM, or getting stuck because of CORS blocks? On the flip side, installing desktop apps can be a hassle because they clutter your storage and leave behind a lot of system "junk"?

We totally get it. That's exactly why we built **Flowork OS**. This isn't just another web app; it's a true *Hybrid OS*.

---

## 🌟 What is Flowork OS?

The concept is simple but *game-changing*. We physically separated the visual interface (UI) from the processing brain:

1. **GUI in the Cloud (Online):** All the sleek, lightweight, and responsive interfaces live on our servers. You'll always get the latest version without the hassle of manual updates.
2. **The Brain in Your PC (Offline/Local):** You just need to run a single, super lightweight `.exe` file in the background. It acts as the "foreman" handling the heavy lifting (Python, Node.js, C++) using your PC's own CPU/GPU power.

These two worlds connect in *real-time* within milliseconds through a secure P2P (Peer-to-Peer) WebSocket tunnel.

---

## 💡 Why You'll Love Flowork

Built with deep research to deliver *native* performance without the headache, here's why Flowork OS stands out:

* 🛡️ **God-Tier Privacy (Zero Data Stored)**
Because the Flowork Engine runs on a local server (*localhost*), essentially turning your own PC into a "Private Server", **we store absolutely no user data**. All your files, *history*, and processed data never even touch our *cloud*. What happens on your PC, stays on your PC!
* 🌐 **"God Mode" (Absolute CORS Bypass)**
Standard web apps can't just casually pull data from other websites. With the Flowork Chrome Extension, your web app gets "god-level" permissions to scrape data or interact cross-domain with zero restrictions.
* 📦 **100% Portable & Zero OS Pollution**
Forget about terminals, `pip install`, or `npm install`. The Flowork Engine automatically downloads all required *libraries* into a hidden folder (`/libs`). Your Windows OS stays perfectly clean, tidy, and pollution-free.
* ⚡ **Anti-Lag & RAM Friendly**
Since the heavy computational lifting is thrown to your local PC, your browser's only job is to render the UI. The result? Apps run buttery smooth at 60fps without choking your browser.
* 🔒 **Safe & Controlled (Smart Kill-Switch)**
Our Engine constantly checks version sync and security with the central server. If there's an outdated version or the server goes into *maintenance*, the connection is instantly cut, and memory is automatically wiped clean. Your system is guaranteed safe.

---

## 🎯 Who Needs This?

Flowork is designed for those who need god-tier performance but want the ease of opening a website:

* **Data Scientists & Automators:** Run heavy, million-line Python scripts on your PC, but monitor the process through a beautiful web dashboard.
* **Creators & Video Editors:** Render high-res videos directly on your local storage. No more wasting bandwidth uploading/downloading gigabytes of files to the *cloud*.
* **Remote Workers & Agencies:** Turn your phone into an absolute *remote control*. Open the web app on your phone, hit "Start," and let your home PC do the heavy data processing.
* **Casual Users:** For those who hate installing a bunch of bulky desktop apps. Just use the 1 Flowork Engine, and you can open hundreds of "Mini Apps" straight from our ecosystem.

---

## 🚀 How to Use (Plug & Play)

You don't need to be a *programmer* or know how to code to enjoy the power of Flowork OS. Just three easy steps:

1. **Download the Engine:** Head over to the [Releases](https://github.com/flowork-dev/Flowork-OS/releases/download/v1.0.1/flowork-engine.exe) tab in this repository, then *download* the latest `flowork-engine.exe` file.
2. **Just Run It:** *Double-click* the downloaded file (No complicated installation process required).
3. **Boom! Ready to Go:** The engine will quietly fire up in your *System Tray* (bottom right of your screen), automatically open your browser, and boom... your local PC and our Web are completely merged!

---
## 🚀 HOW TO BUILD AN OFFLINE APP (LOCAL APP) IN FLOWORK OS

### PART 1: CORE CONCEPTS, HOW IT WORKS & FILE ANATOMY

#### 1. How Does It Work? (The Dual-Engine Architecture)
Offline Apps in Flowork OS stand out from standard web apps because they run on a **Dual-Engine Architecture**. Instead of hogging your browser's memory (RAM), it turns your own PC into a powerhouse "Private Server."

Here is the end-to-end workflow:
1. **The Client (Web UI):** The user clicks the "START" button on the Web interface (`index.html` or `mobile.html`). This button doesn't run heavy computations; it simply triggers a command in `app.js`.
2. **The P2P Tunnel:** `app.js` calls `systemBridge.js`. This script wraps your command into a JSON payload and securely shoots it through a direct P2P WebSocket tunnel to the Golang Engine running in your PC's background (`localhost:5000`).
3. **The Foreman (Golang):** The Golang Engine receives the message, locates your app folder (e.g., `apps/screen-recorder/`), automatically downloads any missing libraries, and executes the Python file (`script.py`).
4. **The Native Executor (Python):** Python reads the command via the "STDIN Pipe", does the heavy lifting (like accessing the file system, rendering videos, or unrestricted web scraping), and prints the result back as a pure JSON object.
5. **Cycle Complete:** Golang catches Python's output and fires it back to the Web UI via the P2P tunnel. The UI receives the data and displays it to the user. All of this happens in mere milliseconds.

#### 2. Folder Anatomy & The Absolute File Laws
Every Local App **must** live inside the `apps/your-app-name/` directory on your PC Engine. Flowork is extremely strict about file structure. If a core file is missing or breaks the rules, the OS will outright refuse to render the app.

Here is the standard boilerplate (using `apps/screen-recorder/` as an example):

```text
apps/screen-recorder/
├── manifest.json       # [REQUIRED] App Identity. Must include "is_local": true.
├── schema.json         # [REQUIRED] The I/O contract between JS UI and Python Backend.
├── requirements.txt    # [IMPORTANT] Python library list. Golang auto-installs these to the /libs folder (Portable Mode).
├── script.py           # [REQUIRED] The backend computational brain (Python/C++/Node).
├── index.html          # [REQUIRED] Desktop UI. Enterprise styling. Zero Logic in HTML.
├── mobile.html         # [REQUIRED] Mobile UI. Edge-to-Edge layout, no blur, Dock Safe Area included.
├── app.js              # [REQUIRED] The frontend brain. Handles clicks, auto-save, i18n, and calls the bridge.
├── systemBridge.js     # [REQUIRED] The P2P Gateway. The only way in or out of the Local Engine.
├── i18n.json           # [REQUIRED] Central dictionary (id & en). No hardcoded text allowed in UI.
├── icon.svg            # [REQUIRED] Sharp, responsive vector icon.
├── readme_en.md        # [REQUIRED] English app guide (rendered as the Info/Lander page).
├── readme_id.md        # [REQUIRED] Indonesian app guide.
├── cover.webp          # [REQUIRED] Desktop cover image (16:9 ratio, e.g., 1280x720 px).
└── cover_mobile.webp   # [REQUIRED] Mobile cover image (9:16 vertical ratio, e.g., 1080x1920 px).

```

**The Golden Rules of Coding:**

* **English Only in Code:** All variable names, functions, and developer comments must be written in English.
* **Zero External Libraries in UI:** CDN assets (like Bootstrap/Tailwind) are strictly forbidden. Use pure Vanilla CSS & JavaScript.

---

### CORE CONFIGURATION & THE PYTHON BRAIN (`script.py`)

#### 1. Absolute Identity: `manifest.json`

This is your app's ID card. The Golang Engine and Web UI read this to decide if your app can be rendered on Desktop or Mobile.
*Strict Rule:* You must include `"is_local": true`, `"desktop": "yes"`, and `"android": "yes"` to pass the OS Security Filter.

Create `apps/screen-recorder/manifest.json`:

```json
{
  "id": "screen-recorder",
  "name": "Local Screen Recorder",
  "version": "1.0.0",
  "description": "Native offline screen recording directly to your local storage via God Mode.",
  "category": "Utility",
  "is_local": true,
  "desktop": "yes",
  "android": "yes",
  "action": {
    "default_popup": "index.html"
  }
}

```

#### 2. The P2P Contract: `schema.json`

This tells the Golang Engine which file to execute when the UI calls for it.
Create `apps/screen-recorder/schema.json`:

```json
{
    "name": "screen-recorder",
    "type": "app",
    "description": "Record PC screen natively using OpenCV and MSS",
    "entry_point": "script.py"
}

```

#### 3. Portable Installation: `requirements.txt`

The ultimate power of the Flowork Engine is its portability. You never have to tell a user to open their terminal and type `pip install`. The Golang Engine reads this file and silently installs everything into a hidden `/libs` folder inside your app directory. No OS pollution!

Create `apps/screen-recorder/requirements.txt`:

```text
mss
numpy
opencv-python

```

#### 4. The Master Brain: `script.py`

This is where God Mode truly happens. Because this connects directly to the P2P WebSocket via Golang, there are **3 Absolute Python Laws** in Flowork OS:

1. **The STDIN Pipe:** Python does not read standard terminal arguments. JSON data from JS is sent via the `sys.stdin.read()` pipe.
2. **Zero Random Prints:** You are STRICTLY FORBIDDEN from using `print("hello")` for standard debugging. The Golang Engine captures `stdout` and throws it back to JS. If you print plain text, the JSON parser will crash. You MUST only print pure JSON Dictionaries.
3. **Absolute UTF-8:** You must reconfigure `stdin` and `stdout` to `utf-8` to prevent weird characters (emojis, kanji) from breaking the bridge.

Here is the full `script.py` utilizing an advanced Lock File technique for continuous looping until JS sends a "Stop" signal.

Create `apps/screen-recorder/script.py`:

```python
import sys
import json
import os
import time
from datetime import datetime

# [ABSOLUTE LAW 1] Encoding protection to prevent P2P Engine crashes
sys.stdin.reconfigure(encoding='utf-8')
sys.stdout.reconfigure(encoding='utf-8')

def main():
    try:
        # [ABSOLUTE LAW 2] Read instructions from JS via STDIN
        input_raw = sys.stdin.read()
        input_data = json.loads(input_raw) if input_raw else {}

        action = input_data.get('action', 'start')

        # Lock File path indicator
        lock_file = os.path.join(os.getcwd(), 'recording.lock')

        # ---------------------------------------------------------
        # SCENARIO A: JS sends a "stop" command
        # ---------------------------------------------------------
        if action == 'stop':
            if os.path.exists(lock_file):
                os.remove(lock_file) # Remove the lock file

            # [ABSOLUTE LAW 3] Reply to JS ONLY via print(json.dumps())
            result = {"status": "success", "message": "Engine stop signal delivered."}
            print(json.dumps({"status": "success", "data": result}))
            sys.exit(0)

        # ---------------------------------------------------------
        # SCENARIO B: JS sends a "start" command
        # ---------------------------------------------------------
        elif action == 'start':
            # Import heavy libraries only on Start to save RAM
            import mss
            import numpy as np
            import cv2

            fps_target = int(input_data.get('fps', 30))

            # Create Lock File to indicate active recording
            with open(lock_file, 'w') as f:
                f.write('RECORDING_ACTIVE')

            output_dir = os.path.join(os.path.expanduser('~'), 'Downloads')
            os.makedirs(output_dir, exist_ok=True)

            filename = f"Flowork_Record_{datetime.now().strftime('%Y%m%d_%H%M%S')}.mp4"
            filepath = os.path.join(output_dir, filename)

            with mss.mss() as sct:
                monitor = sct.monitors[1]
                width = monitor["width"]
                height = monitor["height"]

                fourcc = cv2.VideoWriter_fourcc(*'mp4v')
                out = cv2.VideoWriter(filepath, fourcc, fps_target, (width, height))

                frame_time = 1.0 / fps_target

                # HEAVY LOOP: Records continuously while lock_file exists
                while os.path.exists(lock_file):
                    start_time = time.time()

                    img = np.array(sct.grab(monitor))
                    frame = cv2.cvtColor(img, cv2.COLOR_BGRA2BGR)
                    out.write(frame)

                    # FPS Controller & CPU Cooler
                    elapsed = time.time() - start_time
                    sleep_time = frame_time - elapsed
                    if sleep_time > 0:
                        time.sleep(sleep_time)

                # Absolute Garbage Collection after loop finishes
                out.release()

            # Final result sent back to JS
            result = {
                "status": "success",
                "filepath": filepath,
                "filename": filename,
                "resolution": f"{width}x{height}",
                "fps": fps_target
            }
            print(json.dumps({"status": "success", "data": result}))

    except Exception as e:
        # Return error logs strictly in JSON format
        print(json.dumps({"status": "error", "error": str(e)}))

if __name__ == "__main__":
    main()

```

---

### UI ANATOMY (HTML) & STRICT DICTIONARY

#### 1. Central Dictionary: `i18n.json`

*Strict Rule:* Hardcoding UI text directly in HTML (e.g., `<button>Start</button>`) is strictly forbidden. All text must be called from this dictionary. This ensures your app auto-translates based on the user's OS language settings.

Create `apps/screen-recorder/i18n.json`:

```json
{
  "en": {
    "app_title": "🖥️ Screen Recorder Pro",
    "fps_label": "Frame Rate",
    "start_btn": "START RECORDING",
    "stop_btn": "STOP RECORDING",
    "status_idle": "Ready to record. Engine standby.",
    "status_recording": "Engine is currently capturing screen...",
    "error_p2p": "P2P Engine Connection Error",
    "save_success": "Saved locally to: "
  },
  "id": {
    "app_title": "🖥️ Perekam Layar Pro",
    "fps_label": "Frame Rate",
    "start_btn": "MULAI REKAM",
    "stop_btn": "HENTIKAN REKAMAN",
    "status_idle": "Siap merekam. Mesin siaga.",
    "status_recording": "Sedang merekam layar aktif...",
    "error_p2p": "Koneksi P2P Engine Gagal",
    "save_success": "Tersimpan di: "
  }
}

```

#### 2. Desktop Interface: `index.html`

This UI renders when the app is opened via Web/PC Desktop.
*Absolute HTML Rules:*

* **Zero External Scripts:** No `<link>` or `<script>` to external CDNs. Pure Vanilla CSS only.
* **Zero Logic in HTML:** Native attributes like `onclick="start()"` are prohibited. Use `data-flowork-action="actionName"` instead. `app.js` will handle the event delegation.
* Use `data-i18n="keyName"` to fetch text from `i18n.json`.

Create `apps/screen-recorder/index.html`:
*(Paste your HTML code here, same as the original, just keep the comments in English)*

#### 3. Mobile Interface: `mobile.html`

This UI renders when a user opens the app via Smartphone, turning their phone into an absolute Remote Control for their PC.
*Absolute Mobile Rules:*

* **No Margin Cards:** Must be Edge-to-Edge (full width).
* **Dock Safe Area:** Must include `padding-bottom: 85px;` so OS navigation bars don't overlap the UI.
* **Zero Blur:** `backdrop-filter: blur()` is forbidden. Use solid RGBA for buttery 60fps GPU performance.
* **Anti-Bounce:** Add `overscroll-behavior-y: none;` to the `body` tag for a native app feel.

Create `apps/screen-recorder/mobile.html`:
*(Paste your Mobile HTML code here, same as the original, with English comments)*

---

### P2P TUNNEL & JAVASCRIPT STATE LOGIC

#### 1. The P2P Tunnel: `systemBridge.js`

This is the exclusive gateway for your app to talk to the core Flowork OS.
*Strict Rule:* Do NOT write custom `fetch()` functions targeting localhost. You MUST call `executeEngineTask` from this file to legally bypass CORS via `window.postMessage`.

Create `apps/screen-recorder/systemBridge.js`:

```javascript
export const detectEnvironment = () => {
    return 'web';
};

// God-mode function to send commands to Python
export const executeEngineTask = async (taskName, payload = {}) => {
    const env = detectEnvironment();
    return new Promise((resolve, reject) => {
        // Generate unique ID to prevent data collision
        const taskId = `task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

        const messageHandler = (event) => {
            if (event.data && event.data.type === 'FLOWORK_ENGINE_RESULT' && event.data.taskId === taskId) {
                window.removeEventListener('message', messageHandler);
                if (event.data.error) reject(new Error(event.data.error));
                else resolve(event.data.response);
            }
        };

        // [IMPORTANT] Heavy tasks take time. Timeout extended to 4 Hours (14400000 ms)
        const timeout = setTimeout(() => {
            window.removeEventListener('message', messageHandler);
            reject(new Error(`P2P Engine Timeout.`));
        }, 14400000);

        window.addEventListener('message', messageHandler);

        // Send payload to Core OS (Parent Iframe)
        if (window.parent && window.parent !== window) {
            window.parent.postMessage({
                type: 'FLOWORK_ENGINE_TASK',
                taskId: taskId,
                taskName: taskName, // MUST MATCH APP FOLDER NAME (e.g., "screen-recorder")
                payload: payload,
                environment: env
            }, '*');
        } else {
            clearTimeout(timeout);
            window.removeEventListener('message', messageHandler);
            reject(new Error("Flowork OS Sandbox not detected."));
        }
    });
};

```

#### 2. Interaction Brain & Auto-Save: `app.js`

This is where all frontend logic happens.
*Absolute Logic Rules:*

* **Auto-Save State:** If a user accidentally refreshes the iframe during an active process (like recording), the timer cannot reset to zero. You must save the state in `localStorage`.
* **Zero Default Alerts:** Do not use the browser's native `alert()`. It ruins the OS native feel. Build custom UIs or update text elements instead.
* **Dynamic Dictionary:** On boot, `app.js` MUST detect the active OS language and load `i18n.json`.

Create `apps/screen-recorder/app.js`:
*(Paste your JS code here, same as the original, with English comments)*

---

### 🎉 CONGRATULATIONS! YOUR LOCAL APP IS READY!

For supporting files like `readme_en.md`, `readme_id.md`, `icon.svg`, and `cover.webp`, simply add brief descriptions and images based on your creativity.

**How to Test (Workflow):**

1. Ensure the Golang Engine is running (`go run main.go` or double-click the `.exe`).
2. Open Flowork OS on your PC / Phone.
3. Go to the **Store** page and click the **"My PC Apps"** tab.
4. Your "Screen Recorder Pro" app will appear there.
5. Click the app, hit **START** from your Web/Phone, and boom! Python will silently start recording your screen on your PC (God Mode activated).

---

> 📖 **Want to master Flowork OS development?**
> This is just scratching the surface! For the full, comprehensive guide, advanced techniques, and API documentations, visit our developer blog at:
> **[https://floworkos.com/blog](https://floworkos.com/blog)**

```

