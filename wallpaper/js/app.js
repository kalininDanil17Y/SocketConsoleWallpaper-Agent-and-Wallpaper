import { AgentClient } from "./api.js";
import { AsciiRenderer } from "./ascii.js";
import { TerminalRenderer } from "./terminal.js";
import { applyTheme, getTheme, setCrtEffect } from "./themes.js";

const DEFAULT_PORT = "48771";
const DEFAULT_THEME = "aperture";

const state = {
  port: DEFAULT_PORT,
  online: false,
  metrics: null,
  asciiSource: "",
  themeName: DEFAULT_THEME,
  theme: getTheme(DEFAULT_THEME),
  crtEffect: true
};

const asciiCanvas = document.getElementById("asciiCanvas");
const uiCanvas = document.getElementById("uiCanvas");

const ascii = new AsciiRenderer(asciiCanvas);
const terminal = new TerminalRenderer(uiCanvas, state);
const client = new AgentClient({ port: state.port });

state.theme = applyTheme(state.themeName);
setCrtEffect(state.crtEffect);

client.addEventListener("state", (event) => {
  state.online = Boolean(event.detail.online);
});

client.addEventListener("metrics", (event) => {
  state.metrics = event.detail.status;
});

client.addEventListener("ascii_frame", (event) => {
  state.asciiSource = event.detail.source || "";
  ascii.setFrame(event.detail);
});

client.addEventListener("error_message", () => {
  state.online = false;
});

window.addEventListener("resize", resizeAll);
window.addEventListener("visibilitychange", () => {
  if (!document.hidden) {
    resizeAll();
    client.connect();
  }
});

window.wallpaperPropertyListener = {
  applyUserProperties(properties) {
    if (properties.agentPort) {
      state.port = String(properties.agentPort.value || DEFAULT_PORT);
      client.setPort(state.port);
    }
    if (properties.theme) {
      setTheme(String(properties.theme.value || DEFAULT_THEME));
    }
    if (properties.crtEffect) {
      state.crtEffect = Boolean(properties.crtEffect.value);
      setCrtEffect(state.crtEffect);
    }
  }
};

function setTheme(name) {
  state.themeName = name;
  state.theme = applyTheme(name);
  ascii.draw();
}

function resizeAll() {
  ascii.resize();
  terminal.resize();
}

resizeAll();
terminal.start();
client.connect();
