import { resizeCanvas } from "./ascii.js";
import { activeClockRows, activeTimerRows } from "./timers.js";

export class TerminalRenderer {
  constructor(canvas, state) {
    this.canvas = canvas;
    this.ctx = canvas.getContext("2d", { alpha: true });
    this.state = state;
    this.lastNow = 0;
  }

  resize() {
    resizeCanvas(this.canvas);
  }

  start() {
    const tick = (now) => {
      this.lastNow = now;
      this.draw();
      window.requestAnimationFrame(tick);
    };
    window.requestAnimationFrame(tick);
  }

  draw() {
    const ctx = this.ctx;
    const c = this.canvas;
    const dpr = window.devicePixelRatio || 1;
    const theme = this.state.theme;

    ctx.clearRect(0, 0, c.width, c.height);
    drawGrid(ctx, c.width, c.height, theme, dpr);

    const compact = c.width < 1000 * dpr;
    const panelW = compact ? c.width - 36 * dpr : Math.min(520 * dpr, c.width * 0.36);
    const panelX = compact ? 18 * dpr : 32 * dpr;
    const panelY = compact ? 18 * dpr : 34 * dpr;
    const panelH = compact ? Math.min(c.height - 36 * dpr, 470 * dpr) : c.height - 68 * dpr;

    drawPanel(ctx, panelX, panelY, panelW, panelH, theme, dpr);
    drawHeader(ctx, panelX, panelY, panelW, theme, dpr);

    if (!this.state.online) {
      drawOffline(ctx, panelX, panelY, panelW, theme, this.state.port, dpr);
      drawFooter(ctx, c.width, c.height, theme, this.state, dpr);
      return;
    }

    drawMetrics(ctx, panelX, panelY, panelW, theme, this.state, dpr);
    drawFooter(ctx, c.width, c.height, theme, this.state, dpr);
  }
}

function drawGrid(ctx, width, height, theme, dpr) {
  const step = 48 * dpr;
  ctx.save();
  ctx.strokeStyle = theme.grid;
  ctx.lineWidth = 1 * dpr;
  ctx.globalAlpha = 0.5;
  for (let x = 0; x < width; x += step) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, height);
    ctx.stroke();
  }
  for (let y = 0; y < height; y += step) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(width, y);
    ctx.stroke();
  }
  ctx.restore();
}

function drawPanel(ctx, x, y, w, h, theme, dpr) {
  ctx.save();
  ctx.fillStyle = theme.panel;
  ctx.strokeStyle = theme.grid;
  ctx.lineWidth = 1 * dpr;
  ctx.shadowColor = theme.accent;
  ctx.shadowBlur = 18 * dpr;
  ctx.beginPath();
  ctx.rect(x, y, w, h);
  ctx.fill();
  ctx.shadowBlur = 0;
  ctx.stroke();
  ctx.restore();
}

function drawHeader(ctx, x, y, w, theme, dpr) {
  ctx.save();
  ctx.textBaseline = "top";
  ctx.font = `${14 * dpr}px Consolas, "Cascadia Mono", monospace`;
  ctx.fillStyle = theme.muted;
  ctx.fillText("SOCKET CONSOLE WALLPAPER", x + 22 * dpr, y + 18 * dpr);
  ctx.fillStyle = theme.accent;
  ctx.fillRect(x + 22 * dpr, y + 46 * dpr, Math.max(72 * dpr, w - 44 * dpr), 2 * dpr);
  ctx.restore();
}

function drawMetrics(ctx, x, y, w, theme, state, dpr) {
  const status = state.metrics;
  const visibility = state.metricsVisibility || {};
  const startY = y + 72 * dpr;
  const lineH = 24 * dpr;
  let row = 0;
  const now = new Date();

  ctx.save();
  ctx.textBaseline = "top";
  ctx.font = `${14 * dpr}px Consolas, "Cascadia Mono", monospace`;

  row = drawInfoLine(ctx, x, startY, row, lineH, "OS", status?.system?.os || "unknown", theme, dpr);
  row = drawInfoLine(ctx, x, startY, row, lineH, "HOST", status?.system?.hostname || "unknown", theme, dpr);
  row = drawInfoLine(ctx, x, startY, row, lineH, "UPTIME", formatUptime(currentUptimeSeconds(state)), theme, dpr);

  const clocks = activeClockRows(state.clocks, now);
  if (clocks.length > 0) {
    row++;
    for (const clock of clocks) {
      row = drawInfoLine(ctx, x, startY, row, lineH, trimText(clock.title.toUpperCase(), 8), clock.value, theme, dpr, theme.accent2);
    }
  }

  row++;

  if (visibility.cpu !== false) {
    row = drawMeter(ctx, x, startY, row, lineH, "CPU", status?.cpu?.usagePercent || 0, theme, dpr);
  }
  row = drawMeter(ctx, x, startY, row, lineH, "RAM", status?.memory?.usagePercent || 0, theme, dpr);

  if (visibility.disk !== false) {
    const disks = Array.isArray(status?.disks) ? status.disks : [];
    for (const disk of disks.slice(0, 3)) {
      row = drawMeter(ctx, x, startY, row, lineH, `DISK ${disk.name}`, disk.usagePercent || 0, theme, dpr);
    }
  }

  row++;
  if (visibility.cpu !== false) {
    row = drawInfoLine(ctx, x, startY, row, lineH, "CPU ID", trimText(status?.cpu?.name || "unknown", 30), theme, dpr);
  }
  if (visibility.cores !== false) {
    row = drawInfoLine(ctx, x, startY, row, lineH, "CORES", `${status?.cpu?.cores || 0}C / ${status?.cpu?.threads || 0}T`, theme, dpr);
  }
  if (visibility.ip !== false) {
    row = drawInfoLine(ctx, x, startY, row, lineH, "IP", status?.network?.ipv4 || "n/a", theme, dpr);
  }
  if (visibility.net !== false) {
    row = drawInfoLine(ctx, x, startY, row, lineH, "NET", status?.network?.selectedInterface || "n/a", theme, dpr);
  }

  if (visibility.screen !== false) {
    const screen = Array.isArray(status?.screens) ? status.screens[0] : null;
    row = drawInfoLine(ctx, x, startY, row, lineH, "SCREEN", screen ? `${screen.width}x${screen.height}` : `${window.screen.width}x${window.screen.height}`, theme, dpr);
  }

  const timers = activeTimerRows(state.timers, now);
  if (timers.length > 0) {
    row++;
    ctx.fillStyle = theme.muted;
    ctx.fillText("TIMERS", x + 22 * dpr, startY + row * lineH);
    row++;
    for (const timer of timers) {
      row = drawInfoLine(ctx, x, startY, row, lineH, trimText(timer.title.toUpperCase(), 8), timer.value, theme, dpr, theme.accent);
    }
  }

  ctx.restore();
}

function drawOffline(ctx, x, y, w, theme, port, dpr) {
  ctx.save();
  ctx.textBaseline = "top";
  ctx.font = `${18 * dpr}px Consolas, "Cascadia Mono", monospace`;
  ctx.fillStyle = theme.danger;
  ctx.fillText("AGENT: OFFLINE", x + 22 * dpr, y + 82 * dpr);

  ctx.font = `${15 * dpr}px Consolas, "Cascadia Mono", monospace`;
  ctx.fillStyle = theme.fg;
  ctx.fillText(`Start Socket Console Agent on port ${port}`, x + 22 * dpr, y + 122 * dpr);
  ctx.fillStyle = theme.muted;
  ctx.fillText("Reconnecting automatically...", x + 22 * dpr, y + 152 * dpr);

  ctx.strokeStyle = theme.danger;
  ctx.lineWidth = 1 * dpr;
  ctx.strokeRect(x + 22 * dpr, y + 196 * dpr, w - 44 * dpr, 42 * dpr);
  ctx.fillStyle = theme.danger;
  ctx.fillText("ws://127.0.0.1:" + port + "/api/v1/live", x + 34 * dpr, y + 208 * dpr);
  ctx.restore();
}

function drawFooter(ctx, width, height, theme, state, dpr) {
  const text = `${formatClock(new Date())}  //  ${state.theme.name.toUpperCase()}  //  ${state.online ? "LIVE LINK" : "NO SIGNAL"}`;
  ctx.save();
  ctx.textBaseline = "bottom";
  ctx.textAlign = "right";
  ctx.font = `${13 * dpr}px Consolas, "Cascadia Mono", monospace`;
  ctx.fillStyle = state.online ? theme.muted : theme.danger;
  ctx.fillText(text, width - 26 * dpr, height - 20 * dpr);
  ctx.restore();
}

function drawInfoLine(ctx, x, startY, row, lineH, label, value, theme, dpr, valueColor = theme.fg) {
  const y = startY + row * lineH;
  ctx.fillStyle = theme.muted;
  ctx.fillText(label.padEnd(8, " "), x + 22 * dpr, y);
  ctx.fillStyle = valueColor;
  ctx.fillText(String(value), x + 122 * dpr, y);
  return row + 1;
}

function drawMeter(ctx, x, startY, row, lineH, label, percent, theme, dpr) {
  const y = startY + row * lineH;
  const blocks = 10;
  const filled = Math.max(0, Math.min(blocks, Math.round((percent / 100) * blocks)));
  const bar = "█".repeat(filled) + "░".repeat(blocks - filled);
  ctx.fillStyle = theme.muted;
  ctx.fillText(label.padEnd(8, " "), x + 22 * dpr, y);
  ctx.fillStyle = percent >= 90 ? theme.danger : theme.accent;
  ctx.fillText(bar, x + 122 * dpr, y);
  ctx.fillStyle = theme.fg;
  ctx.fillText(`${Math.round(percent).toString().padStart(3, " ")}%`, x + 284 * dpr, y);
  return row + 1;
}

function currentUptimeSeconds(state) {
  const base = Number(state.metrics?.system?.uptimeSeconds || 0);
  if (!base || !state.metricsReceivedAt) {
    return base;
  }
  return base + Math.floor((Date.now() - state.metricsReceivedAt) / 1000);
}

function formatUptime(totalSeconds) {
  const seconds = Math.max(0, Number(totalSeconds) || 0);
  const days = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (days > 0) {
    return `${days}d ${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  }
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function formatClock(date) {
  return `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}:${String(date.getSeconds()).padStart(2, "0")}`;
}

function trimText(value, max) {
  const text = String(value).trim();
  if (text.length <= max) {
    return text;
  }
  return text.slice(0, Math.max(1, max - 3)) + "...";
}
