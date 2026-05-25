export const THEMES = {
  dark: {
    name: "Dark",
    className: "theme-dark",
    bg: "#050607",
    fg: "#edf6ef",
    muted: "#8e9f98",
    accent: "#6ee7ff",
    accent2: "#ff8d28",
    danger: "#ff5370",
    grid: "rgba(110, 231, 255, 0.13)",
    panel: "rgba(5, 6, 7, 0.72)"
  },
  light: {
    name: "Light",
    className: "theme-light",
    bg: "#e7e3d8",
    fg: "#151815",
    muted: "#5e675f",
    accent: "#0c6b7b",
    accent2: "#b85813",
    danger: "#ad2339",
    grid: "rgba(12, 107, 123, 0.14)",
    panel: "rgba(231, 227, 216, 0.74)"
  },
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
  }
};

const THEME_CLASSES = Object.values(THEMES).map((theme) => theme.className);

export function getTheme(name) {
  return THEMES[name] || THEMES.dark;
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
