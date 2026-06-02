# DownToMate — Smart Local File Organizer 🤖📁

**DownToMate** is a desktop application and background daemon written entirely in Golang. It automatically organizes your folders (like `/Downloads`) in real-time.

Instead of letting your downloads folder become a chaotic mess, DownToMate automatically moves incoming files into designated sub-folders based on intelligent rules using a 3-Tier classification system, including **AI (Google Gemini)**.

---

## ✨ Key Features

- **🖥️ Native Windows GUI:** Comes with an easy-to-use graphical Control Panel. No need to manually edit JSON files.
- **⚡ Real-time & Lightweight:** Uses `fsnotify` to detect new files instantly with *0% CPU usage* while idle.
- **🧠 3-Tier Classification System:**
  1. **Extension Mode:** Move files based on their type (e.g., move `.jpg` to the `Images` folder).
  2. **Keyword Mode:** Move files if their name contains specific keywords (e.g., move files with `assignment` to the `College_Tasks` folder).
  3. **AI Mode (Gemini Flash):** Automatically reads the *content* of text/source code documents using the Google Gemini API to determine the appropriate category.
- **🛡️ Safe & Duplicate Handling:** Automatically handles duplicate file names. If a file with the same name exists, the app appends a number (e.g., `document_1.pdf`) to avoid overwriting your old files.
- **tray System Tray Daemon:** Runs quietly in the background (system tray) without interrupting your workflow.

---

## 🛠️ Prerequisites

Before building this application from source, make sure you have:
1. **[Golang](https://go.dev/dl/)** (Version 1.20+ recommended)
2. **64-bit C Compiler (GCC)** for Windows (Required by the GUI library). Using [MSYS2](https://www.msys2.org/) or [MinGW-w64](https://www.mingw-w64.org/) is highly recommended.

---

## 🚀 Installation & Build

Clone this repository and navigate into the project directory:
```bash
git clone https://github.com/your-username/downtomate.git
cd downtomate
```

Download all dependencies:
```bash
go mod tidy
```

**(Optional but Recommended)** To ensure the GUI is crisp (High DPI aware) and uses modern Windows themes, compile the resource manifest first:
```bash
go install github.com/akavel/rsrc@latest
rsrc -manifest app.manifest -o rsrc.syso
```

Build the application (hiding the black command prompt window):
```bash
go build -ldflags="-H windowsgui" -o DownToMate.exe .
```

---

## ⚙️ AI Configuration (Gemini API)

If you want to use the **AI Monitoring Mode**, you will need a free API Key from Google.

1. Get your API Key from [Google AI Studio](https://aistudio.google.com/).
2. Create a file named `.env` in the same folder as `DownToMate.exe`.
3. Add your key to the `.env` file using this format:
   ```env
   GEMINI_API_KEY=AIzaSy_YOUR_API_KEY_HERE
   ```

*(The application automatically prioritizes the API Key from the `.env` file for your privacy and security so it won't accidentally be committed to GitHub).*

---

## 📖 How to Use

1. Double-click to run **`DownToMate.exe`**.
2. The **Control Panel** will open.
3. In the **Main Settings** section, click `Select Folder...` (Pilih Folder) to specify the directory you want to monitor (e.g., `C:\Users\YourName\Downloads`).
4. In the **Rule Management** section, you can create new rules:
   - Create an Extension rule (e.g., move `.exe` to `Installers`).
   - Create an AI rule (e.g., if a `.txt` or `.c` file contains code, move it to `Source_Code`).
5. Click **▶️ START MONITORING** (Mulai Pantau).
6. Close the window (click the X button). The application will not exit; instead, it minimizes to the **System Tray** (bottom right corner of your screen).

Try downloading or copying a file into your target folder and watch DownToMate's magic at work!

---

## 🏗️ Tech Stack

- **[Golang](https://go.dev/)** - Core programming language.
- **[lxn/walk](https://github.com/lxn/walk)** - Library for Native Windows GUI.
- **[fsnotify](https://github.com/fsnotify/fsnotify)** - Cross-platform file system notifications.
- **[godotenv](https://github.com/joho/godotenv)** - Environment variable management (.env).
- **Google Generative AI API** - The brain behind advanced text processing.

---

*Built to make life more organized.*
