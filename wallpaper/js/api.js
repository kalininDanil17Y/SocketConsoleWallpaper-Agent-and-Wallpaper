export class AgentClient extends EventTarget {
  constructor(options = {}) {
    super();
    this.host = options.host || "127.0.0.1";
    this.port = String(options.port || "48771");
    this.ws = null;
    this.reconnectTimer = 0;
    this.reconnectMs = 1500;
    this.closedByUser = false;
  }

  setPort(port) {
    const normalized = String(port || "48771").trim() || "48771";
    if (this.port === normalized) {
      return;
    }
    this.port = normalized;
    this.reconnect();
  }

  connect() {
    this.closedByUser = false;
    clearTimeout(this.reconnectTimer);

    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.emit("state", { online: false, message: "CONNECTING" });
    const url = `ws://${this.host}:${this.port}/api/v1/live`;
    this.ws = new WebSocket(url);

    this.ws.addEventListener("open", () => {
      this.reconnectMs = 1500;
      this.emit("state", { online: true, message: "ONLINE" });
      this.fetchAsciiFallback();
    });

    this.ws.addEventListener("message", (event) => {
      this.handleMessage(event.data);
    });

    this.ws.addEventListener("close", () => {
      this.ws = null;
      this.emit("state", { online: false, message: "OFFLINE" });
      if (!this.closedByUser) {
        this.scheduleReconnect();
      }
    });

    this.ws.addEventListener("error", () => {
      this.emit("state", { online: false, message: "OFFLINE" });
    });
  }

  disconnect() {
    this.closedByUser = true;
    clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  reconnect() {
    this.disconnect();
    this.closedByUser = false;
    this.connect();
  }

  scheduleReconnect() {
    clearTimeout(this.reconnectTimer);
    const delay = this.reconnectMs;
    this.reconnectMs = Math.min(this.reconnectMs * 1.5, 15000);
    this.reconnectTimer = window.setTimeout(() => this.connect(), delay);
  }

  async fetchAsciiFallback() {
    try {
      const response = await fetch(`http://${this.host}:${this.port}/api/v1/ascii`, { cache: "no-store" });
      if (!response.ok) {
        return;
      }
      const frame = await response.json();
      if (frame && frame.type === "ascii_frame") {
        this.emit("ascii_frame", frame);
      }
    } catch {
      // WebSocket remains the primary transport; this is only a startup fallback.
    }
  }

  handleMessage(raw) {
    let message;
    try {
      message = JSON.parse(raw);
    } catch {
      this.emit("error_message", { message: "Invalid JSON from agent" });
      return;
    }

    if (!message || typeof message.type !== "string") {
      return;
    }

    this.emit(message.type, message);
    if (message.type === "error") {
      this.emit("error_message", message);
    }
  }

  emit(type, detail) {
    this.dispatchEvent(new CustomEvent(type, { detail }));
  }
}
