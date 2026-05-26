# Socket Console Wallpaper

Interactive terminal-style Web Wallpaper for Wallpaper Engine. Live metrics require an optional local companion agent. The wallpaper also works as a standalone NO SIGNAL terminal scene with color ASCII styling, countdown timers, extra UTC clocks, themes, and CRT/VHS effects.

Russian documentation: [README.ru.md](README.ru.md)

## Development Note

This project was developed with AI assistance from Codex GPT-5.5 High.

## Preview

![Socket Console Wallpaper preview](readme-preview.png)

## Security and Transparency

The optional local agent binary has a public [VirusTotal analysis](https://www.virustotal.com/gui/file-analysis/ZWZlZjFkZTQxY2I5ODBkYTVkMTYxZTBiZmE3MzdlNzk6MTc3OTc3Mzc5MA==). At the time of review, 70 of 71 security vendors did not flag `socket-console-agent.exe`; one vendor reported a generic suspicious detection.

All source code is open in this repository. Review, issues, and feedback on any part of the implementation are welcome.

## Agent

The Windows localhost agent lives in `agent/`.

```powershell
cd .\agent
go build -o socket-console-agent.exe .
.\socket-console-agent.exe run
```

### CLI Commands

> **Very important:** `socket-console-agent.exe install` registers the Windows Service with the path to the exact exe you run the command from. The installer does not copy the exe to another directory. Move `socket-console-agent.exe` to a permanent location first, for example `C:\Program Files\Socket Console Agent\socket-console-agent.exe`, then run `install` from there. Do not delete or move that exe after installation, or the service will not start.

| Command | Description |
| --- | --- |
| `socket-console-agent.exe run` | Run the agent in the current console for development and testing. It reads `.\config.json`, creates it when missing, listens on `127.0.0.1`, and stops on `Ctrl+C`. |
| `socket-console-agent.exe install` | Install the agent as a Windows Service with automatic startup. Run from an elevated terminal. |
| `socket-console-agent.exe uninstall` | Remove the Windows Service registration. Stop the service first if it is running. Run from an elevated terminal. |
| `socket-console-agent.exe start` | Start the installed Windows Service. The service must be installed first. |
| `socket-console-agent.exe stop` | Stop the installed Windows Service. |
| `socket-console-agent.exe status` | Print the current Windows Service status: `running`, `stopped`, or `unknown (<code>)`. |
| `socket-console-agent.exe help` | Show general CLI help. |
| `socket-console-agent.exe help <command>` | Show detailed help for a specific command, for example `socket-console-agent.exe help run`. |
| `socket-console-agent.exe run --help` | Show command help through the alternate help flag syntax. |

### Agent Endpoints

Default dev endpoints:

```text
GET  http://127.0.0.1:48771/api/v1/status
GET  http://127.0.0.1:48771/api/v1/config
POST http://127.0.0.1:48771/api/v1/config
GET  http://127.0.0.1:48771/api/v1/interfaces
GET  http://127.0.0.1:48771/api/v1/disks
GET  http://127.0.0.1:48771/api/v1/images
GET  http://127.0.0.1:48771/api/v1/ascii
WS   ws://127.0.0.1:48771/api/v1/live
```

Config paths:

```text
Dev mode:      .\config.json
Service mode:  %ProgramData%\SocketConsoleAgent\config.json
Override:      SOCKET_CONSOLE_AGENT_CONFIG=C:\path\to\config.json
```

## Wallpaper

The Wallpaper Engine web project lives in `wallpaper/`.

For local browser preview:

```powershell
cd .\wallpaper
python -m http.server 48880 --bind 127.0.0.1
```

Then open:

```text
http://127.0.0.1:48880/index.html
```

Wallpaper Engine settings include:

- `theme`: `dark`, `light`, or `aperture`
- `useAgent`: enable connection to the optional local agent for live metrics
- `agentDownloadUrl`: latest release URL for the optional local agent
- `crtEffect`: base CRT scanlines, glow, flicker, and vignette
- `vhsWaveEffect`: rolling old-monitor VHS wave
- `showCpu`, `showRam`, `showGpu`, `showTemperatures`, `showCores`, `showIp`, `showNet`, `showScreen`, `showDisk`
- `showDiskFreeSpace`: display free/total GB next to disk percentages
- `asciiOffsetX`, `asciiOffsetY`
- `timer1Title` ... `timer5Title`
- `timer1Target` ... `timer5Target`
- `clock1Title` ... `clock3Title`
- `clock1Offset` ... `clock3Offset`

Timer target examples:

```text
2026-12-31 23:59
31.12.2026 23:59
31.12
31.12 23:59
```

Clock offset examples:

```text
+7
-4
+5:30
```
