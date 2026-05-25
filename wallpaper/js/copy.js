export const COPY = {
  dark: {
    title: "SOCKET CONSOLE WALLPAPER",
    offline: "AGENT: OFFLINE",
    startAgent: (port) => `Start Socket Console Agent on port ${port}`,
    reconnecting: "Reconnecting automatically...",
    signalOnline: "WEBSOCKET OK",
    signalOffline: "NO SIGNAL",
    timers: "TIMERS"
  },
  light: {
    title: "SOCKET CONSOLE WALLPAPER",
    offline: "AGENT: OFFLINE",
    startAgent: (port) => `Start Socket Console Agent on port ${port}`,
    reconnecting: "Reconnecting automatically...",
    signalOnline: "WEBSOCKET OK",
    signalOffline: "NO SIGNAL",
    timers: "TIMERS"
  },
  aperture: {
    title: "APERTURE SCIENCE TERMINAL",
    offline: "GLaDOS: OFFLINE",
    startAgent: (port) => `Begin local testing protocol on port ${port}`,
    reconnecting: "Re-establishing facility uplink...",
    signalOnline: "FACILITY LINK OK",
    signalOffline: "TEST CHAMBER LOST",
    timers: "TEST QUEUE"
  }
};

export function copyForTheme(themeName) {
  return COPY[themeName] || COPY.dark;
}
