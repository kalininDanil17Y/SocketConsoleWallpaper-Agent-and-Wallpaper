# Socket Console Wallpaper

Project for Wallpaper Engine Web Wallpaper experiments.

## Agent

The Windows localhost agent lives in `agent/`.

```powershell
cd .\agent
go build -o socket-console-agent.exe .
.\socket-console-agent.exe run
```

Available commands:

```text
socket-console-agent.exe run
socket-console-agent.exe install
socket-console-agent.exe uninstall
socket-console-agent.exe start
socket-console-agent.exe stop
socket-console-agent.exe status
socket-console-agent.exe help
socket-console-agent.exe help run
socket-console-agent.exe run --help
```

Default dev endpoint:

```text
http://127.0.0.1:48771/api/v1/status
http://127.0.0.1:48771/api/v1/ascii
ws://127.0.0.1:48771/api/v1/live
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
- `crtEffect`: base CRT scanlines, glow, flicker, and vignette
- `vhsWaveEffect`: rolling old-monitor VHS wave
- `showCpu`, `showRam`, `showCores`, `showIp`, `showNet`, `showScreen`, `showDisk`
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
