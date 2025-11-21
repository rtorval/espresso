# **Espresso ☕**

*Caffeinate your workflow.*  
Espresso is a lightweight Windows system-tray utility designed to keep your computer awake when you need it most. Whether you are reading a long document, giving a presentation, or downloading large files, Espresso prevents your screen from dimming and your system from entering sleep mode.  
Choose your "dosage"—from a quick 10-minute Drop to an infinite shot of Pure Caffeine.

## **💡 Features**

* **The Coffee Menu:** Select from a variety of preset durations tailored to your needs:  
  * 💧 **Drop (10m):** Just a drop, almost no caffeine.  
  * 🍵 **Latte (30m):** Gentle boost.  
  * ☕ **Cappuccino (1h):** Standard session.  
  * ☕ **Americano (3h):** Extended work block.  
  * ⚡ **Espresso (6h) & Lungo (8h):** All-day activity.  
  * 🚀 **Pure Caffeine:** Keep awake indefinitely.  
* **Native & Efficient:** Uses the native Windows API (SetThreadExecutionState) to prevent sleep. No "fake" mouse jiggling or heavy resource usage.  
* **Visual Feedback:** The tray icon changes to reflect the status (Full Cup \= Awake, Empty Cup \= Sleep Allowed).  
* **Live Countdown:** The system tray menu and tooltip display exactly how much time is remaining in your active session.  
* **Non-Intrusive:** Runs quietly in the background. When your session ends, a gentle toast notification informs you that sleep mode is allowed again.  
* **Single-Instance:** Prevents accidental multiple copies from running.

## **🛠️ Installation & Usage**

### **🚀 Download Ready-to-Use Executable**

1. Download the latest Espresso.exe from the [Releases page](https://www.google.com/search?q=https://github.com/rtorval/espresso/releases).  
2. Run the executable. The app immediately starts in the background.  
3. Look for the **Coffee Cup icon** in your system tray.  
4. Right-click to select your mode.

### **⚙️ Build from Source (For Developers)**

To build this project, you need Go 1.21+ installed.

1. **Clone the repository:**  
   git clone \[https://github.com/rtorval/espresso.git\](https://github.com/rtorval/espresso.git)  
   cd espresso

2. **Install Dependencies:**  
   go mod tidy

3. Embed Icons and Windows Resources:  
   Espresso uses embedded resources for the tray icons and metadata.  
   \# Install the tool  
   go install \[github.com/tc-hib/go-winres@latest\](https://github.com/tc-hib/go-winres@latest)

   \# Generate Windows resource files (syso)  
   go-winres make

4. Build the Executable:  
   The \-H=windowsgui flag is crucial to prevent the console window from appearing.  
   go build \-ldflags="-H=windowsgui" \-o Espresso.exe .

## **💻 Technical Details**

Espresso is built entirely in Go and leverages:

* systray — Cross-platform tray integration.  
  github.com/getlantern/systray  
* windows (syscall wrapper) — Specifically SetThreadExecutionState to manage power states.  
  golang.org/x/sys/windows  
* toast — Windows 10+ native toast notifications.  
  github.com/go-toast/toast  
* go-winres — Embeds icons and metadata into the Windows executable.  
  github.com/tc-hib/go-winres

## **⚖️ License**

**© 2025 Rodrigo Toraño Valle**  
This program is free software: you can redistribute it and/or modify  
it under the terms of the GNU General Public License as published by  
the Free Software Foundation, either version 3 of the License, or  
(at your option) any later version.  
This program is distributed in the hope that it will be useful,  
but WITHOUT ANY WARRANTY; without even the implied warranty of  
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the  
GNU General Public License for more details.  
You should have received a copy of the GNU General Public License  
along with this program. If not, see https://www.gnu.org/licenses/.