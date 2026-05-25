const MAX_TIMERS = 5;
const MAX_CLOCKS = 3;

export function timersFromProperties(properties, previous = []) {
  const timers = previous.slice(0, MAX_TIMERS);
  for (let i = 1; i <= MAX_TIMERS; i++) {
    const titleKey = `timer${i}Title`;
    const targetKey = `timer${i}Target`;
    const current = timers[i - 1] || { title: "", target: "" };
    timers[i - 1] = {
      title: properties[titleKey] ? String(properties[titleKey].value || "") : current.title,
      target: properties[targetKey] ? String(properties[targetKey].value || "") : current.target
    };
  }
  return timers;
}

export function clocksFromProperties(properties, previous = []) {
  const clocks = previous.slice(0, MAX_CLOCKS);
  for (let i = 1; i <= MAX_CLOCKS; i++) {
    const titleKey = `clock${i}Title`;
    const offsetKey = `clock${i}Offset`;
    const current = clocks[i - 1] || { title: "", offset: "" };
    clocks[i - 1] = {
      title: properties[titleKey] ? String(properties[titleKey].value || "") : current.title,
      offset: properties[offsetKey] ? String(properties[offsetKey].value || "") : current.offset
    };
  }
  return clocks;
}

export function activeTimerRows(timers, now = new Date()) {
  return timers
    .map((timer, index) => {
      const title = timer.title.trim() || `TIMER ${index + 1}`;
      const target = parseTargetDate(timer.target, now);
      if (!timer.target.trim() || !target) {
        return null;
      }
      return {
        title,
        value: formatCountdown(target, now),
        target
      };
    })
    .filter(Boolean)
    .slice(0, MAX_TIMERS);
}

export function activeClockRows(clocks, now = new Date()) {
  return clocks
    .map((clock, index) => {
      const offsetMinutes = parseUTCOffset(clock.offset);
      if (!clock.offset.trim() || offsetMinutes === null) {
        return null;
      }
      const title = clock.title.trim() || `UTC ${formatOffset(offsetMinutes)}`;
      return {
        title: title || `CLOCK ${index + 1}`,
        value: formatOffsetClock(now, offsetMinutes),
        offset: formatOffset(offsetMinutes)
      };
    })
    .filter(Boolean)
    .slice(0, MAX_CLOCKS);
}

export function parseTargetDate(input, now = new Date()) {
  const text = String(input || "").trim();
  if (!text) {
    return null;
  }

  const iso = text.match(/^(\d{4})[./-](\d{1,2})[./-](\d{1,2})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?)?$/);
  if (iso) {
    return makeDate(Number(iso[1]), Number(iso[2]), Number(iso[3]), Number(iso[4] || 0), Number(iso[5] || 0), Number(iso[6] || 0));
  }

  const dmy = text.match(/^(\d{1,2})[./-](\d{1,2})[./-](\d{4})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?)?$/);
  if (dmy) {
    return makeDate(Number(dmy[3]), Number(dmy[2]), Number(dmy[1]), Number(dmy[4] || 0), Number(dmy[5] || 0), Number(dmy[6] || 0));
  }

  const annual = text.match(/^(\d{1,2})[./-](\d{1,2})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?)?$/);
  if (annual) {
    let target = makeDate(now.getFullYear(), Number(annual[2]), Number(annual[1]), Number(annual[3] || 0), Number(annual[4] || 0), Number(annual[5] || 0));
    if (target && target.getTime() <= now.getTime()) {
      target = makeDate(now.getFullYear() + 1, Number(annual[2]), Number(annual[1]), Number(annual[3] || 0), Number(annual[4] || 0), Number(annual[5] || 0));
    }
    return target;
  }

  return null;
}

export function parseUTCOffset(input) {
  const text = String(input || "").trim();
  const match = text.match(/^([+-])?(\d{1,2})(?::?(\d{2}))?$/);
  if (!match) {
    return null;
  }
  const sign = match[1] === "-" ? -1 : 1;
  const hours = Number(match[2]);
  const minutes = Number(match[3] || 0);
  if (hours > 14 || minutes > 59) {
    return null;
  }
  return sign * (hours * 60 + minutes);
}

function makeDate(year, month, day, hour, minute, second) {
  const date = new Date(year, month - 1, day, hour, minute, second);
  if (
    date.getFullYear() !== year ||
    date.getMonth() !== month - 1 ||
    date.getDate() !== day ||
    date.getHours() !== hour ||
    date.getMinutes() !== minute
  ) {
    return null;
  }
  return date;
}

function formatCountdown(target, now) {
  let diff = target.getTime() - now.getTime();
  if (diff <= 0) {
    return "DONE";
  }

  const totalSeconds = Math.floor(diff / 1000);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (days > 0) {
    return `${days}d ${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
  }
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
}

function formatOffsetClock(now, offsetMinutes) {
  const utc = now.getTime() + now.getTimezoneOffset() * 60000;
  return formatClock(new Date(utc + offsetMinutes * 60000));
}

function formatClock(date) {
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

function formatOffset(offsetMinutes) {
  const sign = offsetMinutes < 0 ? "-" : "+";
  const abs = Math.abs(offsetMinutes);
  const hours = Math.floor(abs / 60);
  const minutes = abs % 60;
  return minutes ? `${sign}${hours}:${pad(minutes)}` : `${sign}${hours}`;
}

function pad(value) {
  return String(value).padStart(2, "0");
}
