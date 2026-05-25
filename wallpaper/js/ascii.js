export class AsciiRenderer {
  constructor(canvas) {
    this.canvas = canvas;
    this.ctx = canvas.getContext("2d", { alpha: true });
    this.frame = null;
    this.layout = null;
  }

  setFrame(frame) {
    this.frame = normalizeFrame(frame);
    this.draw();
  }

  resize() {
    resizeCanvas(this.canvas);
    this.draw();
  }

  clear() {
    const ctx = this.ctx;
    ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
  }

  draw() {
    this.clear();
    if (!this.frame || !this.frame.rows.length) {
      return;
    }

    const ctx = this.ctx;
    const frame = this.frame;
    const dpr = window.devicePixelRatio || 1;
    const width = this.canvas.width;
    const height = this.canvas.height;

    const panelReserve = width >= 1120 * dpr ? Math.min(width * 0.38, 560 * dpr) : 0;
    const pad = Math.max(22 * dpr, width * 0.025);
    const artX = panelReserve + pad;
    const artW = Math.max(240 * dpr, width - artX - pad);
    const artH = height - pad * 2;

    const visualCellRatio = 0.96;
    const fontSizeByW = artW / frame.w / visualCellRatio;
    const fontSizeByH = artH / frame.h / 1.02;
    const fontSize = Math.max(5 * dpr, Math.floor(Math.min(fontSizeByW, fontSizeByH)));

    ctx.font = `${fontSize}px Consolas, "Cascadia Mono", "Courier New", monospace`;
    ctx.textBaseline = "top";
    ctx.textAlign = "left";
    ctx.shadowColor = "rgba(255,255,255,0.11)";
    ctx.shadowBlur = 4 * dpr;

    const naturalCellW = Math.max(1, ctx.measureText("M").width);
    const cellH = Math.max(1, fontSize * 1.02);
    const visualCellW = Math.max(1, fontSize * visualCellRatio);
    const scaleX = visualCellW / naturalCellW;
    const totalW = visualCellW * frame.w;
    const totalH = cellH * frame.h;
    const startX = Math.floor(artX + (artW - totalW) / 2);
    const startY = Math.floor(pad + (artH - totalH) / 2);

    this.layout = { startX, startY, cellW: visualCellW, cellH, fontSize, scaleX };
    ctx.save();
    ctx.translate(startX, startY);
    ctx.scale(scaleX, 1);
    drawFrame(ctx, frame, 0, 0, naturalCellW, cellH);
    ctx.restore();
    ctx.shadowBlur = 0;
  }
}

export function drawFrame(ctx, frame, startX, startY, cellW, cellH) {
  for (let y = 0; y < frame.rows.length; y++) {
    const row = frame.rows[y];
    const text = row.text;
    const fg = row.fg;
    let start = 0;

    while (start < text.length) {
      const colorIndex = fg[start] ?? 0;
      let end = start + 1;

      while (end < text.length && (fg[end] ?? 0) === colorIndex) {
        end++;
      }

      ctx.fillStyle = frame.palette[colorIndex] || "#ffffff";
      ctx.fillText(text.slice(start, end), startX + start * cellW, startY + y * cellH);
      start = end;
    }
  }
}

export function resizeCanvas(canvas) {
  const rect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  const width = Math.max(1, Math.floor(rect.width * dpr));
  const height = Math.max(1, Math.floor(rect.height * dpr));
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width;
    canvas.height = height;
  }
}

function normalizeFrame(frame) {
  const rows = Array.isArray(frame.rows) ? frame.rows : [];
  const width = Number(frame.w) || maxRowWidth(rows) || 90;
  const height = Number(frame.h) || rows.length || 45;
  const palette = Array.isArray(frame.palette) && frame.palette.length ? frame.palette : ["#ffffff"];

  return {
    type: "ascii_frame",
    w: width,
    h: height,
    palette,
    source: frame.source || "",
    rows: rows.map((row) => {
      const text = String(row.text || "").padEnd(width, " ").slice(0, width);
      const fg = Array.isArray(row.fg) ? row.fg.slice(0, width) : [];
      while (fg.length < width) {
        fg.push(0);
      }
      return { text, fg };
    }).slice(0, height)
  };
}

function maxRowWidth(rows) {
  return rows.reduce((max, row) => Math.max(max, String(row.text || "").length), 0);
}
