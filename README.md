# camgo-scrcpy

**Turn your Android phone into a webcam using scrcpy with a beautiful terminal UI.**

camgo-scrcpy is a Go-based TUI application that provides an interactive interface for streaming your Android device's camera to a virtual v4l2 device, making it usable as a webcam in any application.

## Features

- **Interactive TUI** - Beautiful terminal interface using Charmbracelet Bubble Tea
- **Camera Selection** - Choose between front and back camera
- **Resolution Control** - Multiple quality presets (Native, 1080p, 720p, 480p)
- **Automatic Device Detection** - Lists all connected Android devices via ADB
- **Simple Controls** - Keyboard navigation with vim-style shortcuts

## Requirements

### System Dependencies

| Package | Description | Install Command |
|---------|-------------|-----------------|
| `adb` | Android Debug Bridge | `sudo apt install adb` |
| `scrcpy` | Screen mirroring tool (v3+) | [Install from source](https://github.com/Genymobile/scrcpy/blob/master/doc/linux.md#build) |
| `v4l2loopback-dkms` | Virtual camera kernel module | `sudo apt install v4l2loopback-dkms` |

### Runtime Requirements

- Go 1.21+
- Android device with USB debugging enabled
- USB connection (or ADB over Wi-Fi)

## Installation

### From Source

```bash
# Clone the repository
git clone http://72.62.9.100:3000/pym/camgo-scrcpy.git
cd camgo-scrcpy

# Build the application
go build -o camgo-scrcpy .

# Install globally (optional)
sudo mv camgo-scrcpy /usr/local/bin/
```

## Usage

### 1. Enable USB Debugging on Android

1. Go to **Settings > About Phone**
2. Tap **Build Number** 7 times to enable Developer Mode
3. Go to **Settings > Developer Options**
4. Enable **USB Debugging**
5. Connect your phone via USB

### 2. Load the v4l2loopback Module

Load with two virtual devices — one for the Android camera stream, one for OBS Virtual Camera output:

```bash
sudo modprobe v4l2loopback devices=2 exclusive_caps=1,1 card_label="Android Webcam,OBS Virtual Camera"
```

**Make it persistent across reboots:**

```bash
echo 'options v4l2loopback devices=2 exclusive_caps=1,1 card_label="Android Webcam,OBS Virtual Camera"' | sudo tee /etc/modprobe.d/v4l2loopback.conf
echo "v4l2loopback" | sudo tee /etc/modules-load.d/v4l2loopback.conf
```

> `exclusive_caps=1` is required for OBS and other apps to properly detect the virtual devices.

### 3. Run camgo-scrcpy

```bash
./camgo-scrcpy
```

### 4. Use as Webcam

Once streaming, your Android camera is available as a virtual webcam:

- **OBS Studio** - Add a **Video Capture Device** source and select "Android Webcam". OBS Virtual Camera will use the second loopback device automatically.
- **Zoom / Google Meet** - Select "Android Webcam" in camera settings
- **Cheese/Webcamoid** - Works out of the box

> Start camgo-scrcpy and wait for **STREAMING ATIVO** before opening OBS or adding the capture source.

## Controls

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` | Select option |
| `r` | Refresh device list |
| `q` | Quit application |
| `Ctrl+C` | Force quit |

## Resolution Presets

| Preset | Max Size | Best For |
|--------|----------|----------|
| Native | Device native | Maximum quality |
| Full HD | 1920px | Balanced quality/performance |
| HD | 1280px | Low latency streaming |
| SD | 800px | Slow Wi-Fi / older devices |

## How It Works

```
┌─────────────┐     ADB      ┌─────────────┐    scrcpy     ┌──────────────┐
│   Android   │◄────────────►│    host     │──────────────►│ v4l2loopback │
│   Device    │   USB/WiFi   │   machine   │   H.264       │ /dev/video10 │
└─────────────┘              └─────────────┘               └──────────────┘
                                                                   │
                                                                   ▼
                                                           ┌──────────────┐
                                                           │   Webcam     │
                                                           │   Apps (OBS, │
                                                           │   Zoom, etc) │
                                                           └──────────────┘
```

1. **Device Detection**: The app uses `adb devices` to find connected Android devices
2. **Camera Selection**: Communicates with scrcpy to select front/back camera
3. **Video Encoding**: Uses H.264 hardware encoding on the Android device
4. **Virtual Device**: Streams to v4l2loopback kernel module
5. **Webcam Access**: Any application can now use the phone camera

## Troubleshooting

### "No device found"

1. Check USB connection: `adb devices`
2. Authorize USB debugging on your phone (tap "Allow" when prompted)
3. Try: `adb kill-server && adb devices`

### "scrcpy: device not found"

- Ensure scrcpy version 3.0+ is installed (older versions don't support camera source)

### "v4l2-sink failed" / "no loopback device found"

- Make sure v4l2loopback module is loaded: `ls /dev/video*`
- If missing: `sudo modprobe v4l2loopback devices=2 exclusive_caps=1,1 card_label="Android Webcam,OBS Virtual Camera"`

### OBS "Failed to start virtual camera"

- Load v4l2loopback with `devices=2` so OBS has its own dedicated loopback device (see step 2)
- Make sure camgo-scrcpy is running before trying to start OBS Virtual Camera

### High latency

- Use lower resolution preset (HD or SD)
- Use USB instead of Wi-Fi for ADB
- Ensure H.264 encoding is supported on your device

## Configuration

Currently, the app uses sensible defaults. Future versions may support:

- Custom v4l2 device path
- Preferred camera (front/back)
- Custom resolution
- Audio passthrough

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [scrcpy](https://github.com/Genymobile/scrcpy) - Video mirroring without root
- [Charmbracelet](https://charm.sh/) - TUI framework and utilities
- [v4l2loopback](https://github.com/umlaeute/v4l2loopback) - Virtual video devices for Linux
