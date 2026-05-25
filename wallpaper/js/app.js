import { AgentClient } from "./api.js";
import { AsciiRenderer } from "./ascii.js";
import { TerminalRenderer } from "./terminal.js";
import { clocksFromProperties, timersFromProperties } from "./timers.js";
import { applyTheme, getTheme, setCrtEffect, setVhsWaveEffect } from "./themes.js";

const DEFAULT_PORT = "48771";
const DEFAULT_THEME = "dark";

const state = {
  port: DEFAULT_PORT,
  online: false,
  metrics: null,
  metricsReceivedAt: 0,
  asciiSource: "",
  asciiOffsetX: 0,
  asciiOffsetY: 0,
  themeName: DEFAULT_THEME,
  theme: getTheme(DEFAULT_THEME),
  crtEffect: true,
  vhsWaveEffect: true,
  metricsVisibility: {
    cpu: true,
    ram: true,
    gpu: true,
    temperatures: true,
    cores: true,
    ip: true,
    net: true,
    screen: true,
    disk: true
  },
  timers: Array.from({ length: 5 }, () => ({ enabled: false, title: "", target: "" })),
  clocks: Array.from({ length: 3 }, () => ({ enabled: false, title: "", offset: "" }))
};

const asciiCanvas = document.getElementById("asciiCanvas");
const uiCanvas = document.getElementById("uiCanvas");

const ascii = new AsciiRenderer(asciiCanvas);
const terminal = new TerminalRenderer(uiCanvas, state);
const client = new AgentClient({ port: state.port });

state.theme = applyTheme(state.themeName);
setCrtEffect(state.crtEffect);
setVhsWaveEffect(state.vhsWaveEffect);

client.addEventListener("state", (event) => {
  state.online = Boolean(event.detail.online);
  if (!state.online) {
    ascii.drawNoSignal(state.theme);
  } else {
    ascii.draw();
  }
});

client.addEventListener("metrics", (event) => {
  state.metrics = event.detail.status;
  state.metricsReceivedAt = Date.now();
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
    if (properties.vhsWaveEffect) {
      state.vhsWaveEffect = Boolean(properties.vhsWaveEffect.value);
      setVhsWaveEffect(state.vhsWaveEffect);
    }
    applyMetricVisibility(properties);
    applyAsciiLayout(properties);
    state.timers = timersFromProperties(properties, state.timers);
    state.clocks = clocksFromProperties(properties, state.clocks);
  }
};

function setTheme(name) {
  state.themeName = name;
  state.theme = applyTheme(name);
  if (state.online) {
    ascii.draw();
  } else {
    ascii.drawNoSignal(state.theme);
  }
}

function resizeAll() {
  ascii.resize();
  terminal.resize();
}

function applyMetricVisibility(properties) {
  const map = {
    showCpu: "cpu",
    showRam: "ram",
    showGpu: "gpu",
    showTemperatures: "temperatures",
    showCores: "cores",
    showIp: "ip",
    showNet: "net",
    showScreen: "screen",
    showDisk: "disk"
  };
  for (const [propertyName, stateName] of Object.entries(map)) {
    if (properties[propertyName]) {
      state.metricsVisibility[stateName] = Boolean(properties[propertyName].value);
    }
  }
}

function applyAsciiLayout(properties) {
  let changed = false;
  if (properties.asciiOffsetX) {
    state.asciiOffsetX = Number(properties.asciiOffsetX.value) || 0;
    changed = true;
  }
  if (properties.asciiOffsetY) {
    state.asciiOffsetY = Number(properties.asciiOffsetY.value) || 0;
    changed = true;
  }
  if (changed) {
    ascii.setOffset(state.asciiOffsetX, state.asciiOffsetY);
    if (!state.online) {
      ascii.drawNoSignal(state.theme);
    }
  }
}

resizeAll();
terminal.start();
client.connect();
