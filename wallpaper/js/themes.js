export const THEMES = {
  aperture: {
    name: "Aperture Science",
    className: "theme-aperture",
    bg: "#050607",
    fg: "#edf6ef",
    muted: "#8e9f98",
    accent: "#ff8d28",
    accent2: "#6ee7ff",
    danger: "#ff5370",
    grid: "rgba(255, 141, 40, 0.15)",
    panel: "rgba(5, 6, 7, 0.72)"
  },
  windows: {
    name: "Windows Terminal",
    className: "theme-windows",
    bg: "#07111f",
    fg: "#d7e7ff",
    muted: "#83a2c7",
    accent: "#4cc2ff",
    accent2: "#a78bfa",
    danger: "#ff5c8a",
    grid: "rgba(76, 194, 255, 0.15)",
    panel: "rgba(7, 17, 31, 0.72)"
  },
  amber: {
    name: "Amber CRT",
    className: "theme-amber",
    bg: "#0b0702",
    fg: "#ffd48a",
    muted: "#aa7d42",
    accent: "#ffb000",
    accent2: "#ffe2a8",
    danger: "#ff684f",
    grid: "rgba(255, 176, 0, 0.16)",
    panel: "rgba(11, 7, 2, 0.74)"
  },
  matrix: {
    name: "Matrix",
    className: "theme-matrix",
    bg: "#010602",
    fg: "#b7ffbf",
    muted: "#4c8a57",
    accent: "#00ff66",
    accent2: "#eaffea",
    danger: "#ff4d6d",
    grid: "rgba(0, 255, 102, 0.16)",
    panel: "rgba(1, 6, 2, 0.72)"
  },
  mono: {
    name: "Minimal Mono",
    className: "theme-mono",
    bg: "#060606",
    fg: "#eeeeee",
    muted: "#8f8f8f",
    accent: "#ffffff",
    accent2: "#c8c8c8",
    danger: "#ff6f6f",
    grid: "rgba(255, 255, 255, 0.12)",
    panel: "rgba(6, 6, 6, 0.76)"
  }
};

const THEME_CLASSES = Object.values(THEMES).map((theme) => theme.className);

export function getTheme(name) {
  return THEMES[name] || THEMES.aperture;
}

export function applyTheme(name) {
  const theme = getTheme(name);
  document.body.classList.remove(...THEME_CLASSES);
  document.body.classList.add(theme.className);
  return theme;
}

export function setCrtEffect(enabled) {
  document.body.classList.toggle("crt-enabled", Boolean(enabled));
}
