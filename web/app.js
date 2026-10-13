// GoodAI Monitor - Web Dashboard Application Logic (100% English)

document.addEventListener("DOMContentLoaded", () => {
  initTheme();
  initClock();
  pollHealth();
  setInterval(pollHealth, 5000);
});

// Theme Management
function initTheme() {
  const toggleBtn = document.getElementById("theme-toggle");
  const themeIcon = document.getElementById("theme-icon");
  const savedTheme = localStorage.getItem("goodai-theme") || "theme-dark";

  applyTheme(savedTheme);

  toggleBtn.addEventListener("click", () => {
    const isDark = document.body.classList.contains("theme-dark");
    const nextTheme = isDark ? "theme-light" : "theme-dark";
    applyTheme(nextTheme);
    localStorage.setItem("goodai-theme", nextTheme);
  });

  function applyTheme(theme) {
    document.body.classList.remove("theme-dark", "theme-light");
    document.body.classList.add(theme);
    if (theme === "theme-light") {
      themeIcon.innerHTML = "&#9790;"; // Moon symbol for switching to dark
    } else {
      themeIcon.innerHTML = "&#9788;"; // Sun symbol for switching to light
    }
  }
}

// Clock Display (NZDT / UTC)
function initClock() {
  const clockEl = document.getElementById("clock-display");
  function updateClock() {
    const now = new Date();
    const nzTimeStr = new Intl.DateTimeFormat("en-NZ", {
      timeZone: "Pacific/Auckland",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false
    }).format(now);
    clockEl.textContent = `${nzTimeStr} NZDT`;
  }
  updateClock();
  setInterval(updateClock, 1000);
}

// Health Polling
async function pollHealth() {
  const lastPollEl = document.getElementById("last-poll-time");
  try {
    const response = await fetch("/healthz", { cache: "no-store" });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const data = await response.json();
    updateUIWithHealthData(data);
    lastPollEl.textContent = new Date().toLocaleTimeString("en-NZ", { hour12: false });
  } catch (error) {
    console.warn("Failed to poll /healthz:", error);
    markSystemDegraded("Failed to connect to health endpoint");
    lastPollEl.textContent = "Connection Error";
  }
}

function updateUIWithHealthData(data) {
  // Update Uptime
  const uptimeEl = document.getElementById("uptime-value");
  if (uptimeEl && typeof data.uptime_seconds === "number") {
    uptimeEl.textContent = formatUptime(data.uptime_seconds);
  }

  // Update Memory RSS
  const rssEl = document.getElementById("rss-value");
  const rssBar = document.getElementById("rss-progress");
  const rssSubtext = document.getElementById("rss-subtext");
  if (rssEl && typeof data.memory_rss_bytes === "number") {
    const mib = data.memory_rss_bytes / (1024 * 1024);
    rssEl.textContent = `${mib.toFixed(1)} MiB`;

    const budgetMib = 20.0;
    const pct = Math.min(Math.round((mib / budgetMib) * 100), 100);
    if (rssBar) {
      rssBar.style.width = `${pct}%`;
      if (pct > 90) {
        rssBar.style.backgroundColor = "var(--color-down)";
      } else if (pct > 75) {
        rssBar.style.backgroundColor = "var(--color-degraded)";
      } else {
        rssBar.style.backgroundColor = "var(--color-up)";
      }
    }
    if (rssSubtext) {
      rssSubtext.textContent = `${pct}% of 20.0 MiB budget`;
    }
  }

  // Update Database Connection
  const dbBadge = document.getElementById("db-badge");
  if (dbBadge) {
    if (data.database === "connected") {
      dbBadge.className = "badge-subtle badge-up";
      dbBadge.textContent = "Connected";
    } else {
      dbBadge.className = "badge-subtle badge-down";
      dbBadge.textContent = "Offline";
    }
  }

  // Update Goroutines & Heap
  const goroutinesEl = document.getElementById("goroutines-value");
  const heapBadge = document.getElementById("heap-badge");
  if (goroutinesEl && typeof data.go_goroutine_count === "number") {
    goroutinesEl.textContent = data.go_goroutine_count;
  }
  if (heapBadge && typeof data.go_heap_alloc_bytes === "number") {
    const heapMib = (data.go_heap_alloc_bytes / (1024 * 1024)).toFixed(1);
    heapBadge.textContent = `${heapMib} MiB Heap`;
  }

  // Update System Status Banner
  const bannerBadge = document.getElementById("banner-badge");
  const bannerText = document.getElementById("banner-status-text");
  const bannerHeading = document.getElementById("banner-heading");
  const bannerSubtext = document.getElementById("banner-subtext");
  const brandIndicator = document.getElementById("brand-indicator");

  if (data.status === "ok" && data.database === "connected") {
    if (bannerBadge) bannerBadge.className = "banner-status-badge badge-up";
    if (bannerText) bannerText.textContent = "OPERATIONAL";
    if (bannerHeading) bannerHeading.textContent = "All Systems Functioning Normally";
    if (bannerSubtext) bannerSubtext.textContent = "Continuous background probing active with zero detected outages.";
    if (brandIndicator) brandIndicator.className = "status-indicator-dot online";
  } else {
    markSystemDegraded("One or more subsystems reported an issue");
  }
}

function markSystemDegraded(reason) {
  const bannerBadge = document.getElementById("banner-badge");
  const bannerText = document.getElementById("banner-status-text");
  const bannerHeading = document.getElementById("banner-heading");
  const bannerSubtext = document.getElementById("banner-subtext");
  const brandIndicator = document.getElementById("brand-indicator");

  if (bannerBadge) bannerBadge.className = "banner-status-badge badge-degraded";
  if (bannerText) bannerText.textContent = "DEGRADED";
  if (bannerHeading) bannerHeading.textContent = "Telemetry Degraded";
  if (bannerSubtext) bannerSubtext.textContent = reason;
  if (brandIndicator) brandIndicator.className = "status-indicator-dot offline";
}

function formatUptime(seconds) {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;

  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}
