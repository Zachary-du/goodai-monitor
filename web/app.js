// GoodAI Monitor - Web Dashboard Application Logic (100% English)

let currentUser = null;
let monitorPollInterval = null;

document.addEventListener("DOMContentLoaded", () => {
  initTheme();
  initClock();
  initAuth();
  initMonitors();
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

// Clock Display (NZDT / Auckland)
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

// Authentication & Modal Management
function initAuth() {
  const guestNav = document.getElementById("guest-nav");
  const userNav = document.getElementById("user-nav");
  const userAvatar = document.getElementById("user-avatar");
  const userDisplayName = document.getElementById("user-display-name");

  const loginModal = document.getElementById("login-modal");
  const registerModal = document.getElementById("register-modal");

  const btnOpenLogin = document.getElementById("btn-open-login");
  const btnOpenRegister = document.getElementById("btn-open-register");
  const btnCloseLogin = document.getElementById("btn-close-login");
  const btnCloseRegister = document.getElementById("btn-close-register");
  const btnSwitchToRegister = document.getElementById("btn-switch-to-register");
  const btnSwitchToLogin = document.getElementById("btn-switch-to-login");
  const btnLogout = document.getElementById("btn-logout");

  const loginForm = document.getElementById("login-form");
  const registerForm = document.getElementById("register-form");
  const loginAlert = document.getElementById("login-alert");
  const registerAlert = document.getElementById("register-alert");
  const loginSubmitBtn = document.getElementById("login-submit-btn");
  const registerSubmitBtn = document.getElementById("register-submit-btn");

  // Initial session verification
  checkSession();

  // Modal Open / Close Triggers
  btnOpenLogin.addEventListener("click", () => openModal(loginModal));
  btnOpenRegister.addEventListener("click", () => openModal(registerModal));
  btnCloseLogin.addEventListener("click", () => closeModal(loginModal));
  btnCloseRegister.addEventListener("click", () => closeModal(registerModal));

  btnSwitchToRegister.addEventListener("click", () => {
    closeModal(loginModal);
    openModal(registerModal);
  });

  btnSwitchToLogin.addEventListener("click", () => {
    closeModal(registerModal);
    openModal(loginModal);
  });

  // Close modals when clicking backdrop
  [loginModal, registerModal].forEach((modal) => {
    modal.addEventListener("click", (e) => {
      if (e.target === modal) {
        closeModal(modal);
      }
    });
  });

  // Close modals on Escape key
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      closeModal(loginModal);
      closeModal(registerModal);
      const addModal = document.getElementById("add-monitor-modal");
      const editModal = document.getElementById("edit-monitor-modal");
      const deleteModal = document.getElementById("delete-monitor-modal");
      if (addModal) closeModal(addModal);
      if (editModal) closeModal(editModal);
      if (deleteModal) closeModal(deleteModal);
    }
  });

  // Login Form Submission
  loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    clearAlert(loginAlert);

    const email = document.getElementById("login-email").value.trim();
    const password = document.getElementById("login-password").value;

    if (!email || !password) {
      showAlert(loginAlert, "Please provide both email and password.", "error");
      return;
    }

    loginSubmitBtn.disabled = true;
    loginSubmitBtn.textContent = "Signing In...";

    try {
      const response = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password })
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Authentication failed");
      }

      setAuthenticatedState(data.user);
      loginForm.reset();
      closeModal(loginModal);
      showToast("Signed in successfully.", "success");
    } catch (err) {
      showAlert(loginAlert, err.message, "error");
    } finally {
      loginSubmitBtn.disabled = false;
      loginSubmitBtn.textContent = "Sign In";
    }
  });

  // Register Form Submission
  registerForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    clearAlert(registerAlert);

    const name = document.getElementById("register-name").value.trim();
    const email = document.getElementById("register-email").value.trim();
    const password = document.getElementById("register-password").value;

    if (password.length < 8) {
      showAlert(registerAlert, "Password must be at least 8 characters long.", "error");
      return;
    }

    registerSubmitBtn.disabled = true;
    registerSubmitBtn.textContent = "Creating Account...";

    try {
      const response = await fetch("/api/v1/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, email, password })
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Failed to create account");
      }

      setAuthenticatedState(data.user);
      registerForm.reset();
      closeModal(registerModal);
      showToast("Account created successfully. Welcome to GoodAI Monitor!", "success");
    } catch (err) {
      showAlert(registerAlert, err.message, "error");
    } finally {
      registerSubmitBtn.disabled = false;
      registerSubmitBtn.textContent = "Create Account";
    }
  });

  // Logout Trigger
  btnLogout.addEventListener("click", async () => {
    try {
      await fetch("/api/v1/auth/logout", { method: "POST" });
    } catch (err) {
      console.warn("Logout request failed:", err);
    }
    setGuestState();
    showToast("Signed out successfully.", "info");
  });

  async function checkSession() {
    try {
      const response = await fetch("/api/v1/auth/me", { cache: "no-store" });
      if (response.ok) {
        const data = await response.json();
        if (data.user) {
          setAuthenticatedState(data.user);
          return;
        }
      }
    } catch (err) {
      // Session absent or expired
    }
    setGuestState();
  }

  function setAuthenticatedState(user) {
    currentUser = user;
    guestNav.style.display = "none";
    userNav.style.display = "flex";

    const displayName = user.name || user.email.split("@")[0];
    userDisplayName.textContent = displayName;
    userAvatar.textContent = (displayName.charAt(0) || "U").toUpperCase();

    // Show authenticated monitor controls
    const btnOpenAdd = document.getElementById("btn-open-add-monitor");
    const quotaDisplay = document.getElementById("monitor-quota-display");
    if (btnOpenAdd) btnOpenAdd.style.display = "inline-flex";
    if (quotaDisplay) quotaDisplay.style.display = "inline-block";

    loadMonitors();
    startMonitorPolling();
  }

  function setGuestState() {
    currentUser = null;
    guestNav.style.display = "flex";
    userNav.style.display = "none";

    // Hide authenticated monitor controls
    const btnOpenAdd = document.getElementById("btn-open-add-monitor");
    const quotaDisplay = document.getElementById("monitor-quota-display");
    if (btnOpenAdd) btnOpenAdd.style.display = "none";
    if (quotaDisplay) quotaDisplay.style.display = "none";

    stopMonitorPolling();
    renderGuestMonitors();
  }
}

// Monitor Target Management
function initMonitors() {
  const addModal = document.getElementById("add-monitor-modal");
  const editModal = document.getElementById("edit-monitor-modal");
  const deleteModal = document.getElementById("delete-monitor-modal");

  const btnOpenAdd = document.getElementById("btn-open-add-monitor");
  const btnCloseAdd = document.getElementById("btn-close-add-monitor");
  const btnCloseEdit = document.getElementById("btn-close-edit-monitor");
  const btnCloseDelete = document.getElementById("btn-close-delete-modal");
  const btnCancelDelete = document.getElementById("btn-cancel-delete");
  const btnConfirmDelete = document.getElementById("btn-confirm-delete");

  const addForm = document.getElementById("add-monitor-form");
  const editForm = document.getElementById("edit-monitor-form");
  const addAlert = document.getElementById("add-monitor-alert");
  const editAlert = document.getElementById("edit-monitor-alert");
  const addSubmitBtn = document.getElementById("add-monitor-submit-btn");
  const editSubmitBtn = document.getElementById("edit-monitor-submit-btn");

  // Modal backdrop close listeners
  [addModal, editModal, deleteModal].forEach((modal) => {
    if (modal) {
      modal.addEventListener("click", (e) => {
        if (e.target === modal) closeModal(modal);
      });
    }
  });

  // Modal open & close buttons
  if (btnOpenAdd) {
    btnOpenAdd.addEventListener("click", () => {
      clearAlert(addAlert);
      addForm.reset();
      openModal(addModal);
    });
  }
  if (btnCloseAdd) btnCloseAdd.addEventListener("click", () => closeModal(addModal));
  if (btnCloseEdit) btnCloseEdit.addEventListener("click", () => closeModal(editModal));
  if (btnCloseDelete) btnCloseDelete.addEventListener("click", () => closeModal(deleteModal));
  if (btnCancelDelete) btnCancelDelete.addEventListener("click", () => closeModal(deleteModal));

  // Add Monitor Form Submission
  if (addForm) {
    addForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      clearAlert(addAlert);

      const name = document.getElementById("add-monitor-name").value.trim();
      const url = document.getElementById("add-monitor-url").value.trim();
      const interval = parseInt(document.getElementById("add-monitor-interval").value, 10);
      const timeout = parseInt(document.getElementById("add-monitor-timeout").value, 10);
      const statusCode = parseInt(document.getElementById("add-monitor-status-code").value, 10);
      const keyword = document.getElementById("add-monitor-keyword").value.trim();

      // Client-side URL validation
      if (!url.startsWith("http://") && !url.startsWith("https://")) {
        showAlert(addAlert, "Target URL must start with http:// or https://", "error");
        return;
      }

      addSubmitBtn.disabled = true;
      addSubmitBtn.textContent = "Validating & Creating...";

      try {
        const response = await fetch("/api/v1/monitors", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            name,
            url,
            interval_seconds: interval,
            timeout_seconds: timeout,
            expected_status_code: statusCode,
            keyword
          })
        });

        const data = await response.json();
        if (!response.ok) {
          throw new Error(data.error || "Failed to create monitor target");
        }

        closeModal(addModal);
        addForm.reset();
        showToast("Monitor created successfully.", "success");
        loadMonitors();
      } catch (err) {
        showAlert(addAlert, err.message, "error");
      } finally {
        addSubmitBtn.disabled = false;
        addSubmitBtn.textContent = "Create Monitor";
      }
    });
  }

  // Edit Monitor Form Submission
  if (editForm) {
    editForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      clearAlert(editAlert);

      const id = document.getElementById("edit-monitor-id").value;
      const name = document.getElementById("edit-monitor-name").value.trim();
      const url = document.getElementById("edit-monitor-url").value.trim();
      const interval = parseInt(document.getElementById("edit-monitor-interval").value, 10);
      const timeout = parseInt(document.getElementById("edit-monitor-timeout").value, 10);
      const statusCode = parseInt(document.getElementById("edit-monitor-status-code").value, 10);
      const keyword = document.getElementById("edit-monitor-keyword").value.trim();

      if (!url.startsWith("http://") && !url.startsWith("https://")) {
        showAlert(editAlert, "Target URL must start with http:// or https://", "error");
        return;
      }

      editSubmitBtn.disabled = true;
      editSubmitBtn.textContent = "Saving...";

      try {
        const response = await fetch(`/api/v1/monitors/${id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            name,
            url,
            interval_seconds: interval,
            timeout_seconds: timeout,
            expected_status_code: statusCode,
            keyword
          })
        });

        const data = await response.json();
        if (!response.ok) {
          throw new Error(data.error || "Failed to update monitor");
        }

        closeModal(editModal);
        showToast("Monitor updated successfully.", "success");
        loadMonitors();
      } catch (err) {
        showAlert(editAlert, err.message, "error");
      } finally {
        editSubmitBtn.disabled = false;
        editSubmitBtn.textContent = "Save Changes";
      }
    });
  }

  // Delete Monitor Confirmation
  if (btnConfirmDelete) {
    btnConfirmDelete.addEventListener("click", async () => {
      const id = document.getElementById("delete-monitor-id").value;
      btnConfirmDelete.disabled = true;
      btnConfirmDelete.textContent = "Deleting...";

      try {
        const response = await fetch(`/api/v1/monitors/${id}`, {
          method: "DELETE"
        });

        const data = await response.json();
        if (!response.ok) {
          throw new Error(data.error || "Failed to delete monitor");
        }

        closeModal(deleteModal);
        showToast("Monitor deleted successfully.", "success");
        loadMonitors();
      } catch (err) {
        showToast(err.message, "error");
      } finally {
        btnConfirmDelete.disabled = false;
        btnConfirmDelete.textContent = "Delete Monitor";
      }
    });
  }
}

// Load and Render Monitors
async function loadMonitors() {
  if (!currentUser) return;

  try {
    const response = await fetch("/api/v1/monitors", { cache: "no-store" });
    if (!response.ok) {
      if (response.status === 401) {
        return;
      }
      throw new Error(`HTTP ${response.status}`);
    }

    const data = await response.json();
    const monitors = data.monitors || [];
    renderMonitorsTable(monitors);
    updateQuotaBadge(monitors.length);
  } catch (err) {
    console.warn("Failed to load monitors:", err);
  }
}

function renderMonitorsTable(monitors) {
  const tbody = document.getElementById("monitors-table-body");
  const countBadge = document.getElementById("monitor-count");

  if (countBadge) {
    countBadge.textContent = `${monitors.length} ${monitors.length === 1 ? "Target" : "Targets"}`;
  }

  if (!tbody) return;

  if (monitors.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="7" style="text-align: center; padding: 2.5rem 1rem;">
          <div style="color: var(--text-secondary); margin-bottom: 0.5rem; font-size: 0.9375rem; font-weight: 500;">
            No monitoring targets configured yet.
          </div>
          <div style="color: var(--text-muted); font-size: 0.8125rem;">
            Click <strong>+ Add Target</strong> above to start monitoring your websites and APIs.
          </div>
        </td>
      </tr>
    `;
    return;
  }

  tbody.innerHTML = monitors.map((m) => {
    const statusBadgeClass = getBadgeClassForStatus(m.status);
    const latencyDisplay = m.last_latency_ms > 0 ? `${m.last_latency_ms} ms` : "--";
    const uptimeDisplay = typeof m.uptime_percent === "number" ? `${m.uptime_percent.toFixed(1)}%` : "100.0%";
    const isPaused = m.status === "PAUSED";
    const pauseLabel = isPaused ? "&#9658; Resume" : "&#10074;&#10074; Pause";
    const pauseTitle = isPaused ? "Resume probing" : "Pause probing";

    return `
      <tr data-id="${m.id}">
        <td>
          <div class="service-cell">
            <span class="service-name">${escapeHTML(m.name)}</span>
            ${m.keyword ? `<span class="service-tags">Assert: &quot;${escapeHTML(m.keyword)}&quot;</span>` : `<span class="service-tags">Expected: HTTP ${m.expected_status_code}</span>`}
          </div>
        </td>
        <td><code class="url-badge" title="${escapeHTML(m.url)}">${truncateText(m.url, 36)}</code></td>
        <td>${m.interval_seconds}s</td>
        <td><span style="font-family: var(--font-mono); font-size: 0.8125rem;">${latencyDisplay}</span></td>
        <td><span style="font-family: var(--font-mono); font-size: 0.8125rem; font-weight: 600; color: ${m.uptime_percent >= 99 ? 'var(--color-up)' : 'var(--color-degraded)'}">${uptimeDisplay}</span></td>
        <td><span class="badge ${statusBadgeClass}">${m.status}</span></td>
        <td style="text-align: right;">
          <div class="action-btn-group">
            <button class="btn-action btn-action-pause" onclick="togglePauseMonitor(${m.id})" title="${pauseTitle}">
              ${pauseLabel}
            </button>
            <button class="btn-action" onclick="openEditMonitorModal(${m.id})" title="Edit target settings">
              &#9998; Edit
            </button>
            <button class="btn-action btn-action-delete" onclick="openDeleteMonitorModal(${m.id}, '${escapeHTML(m.name)}')" title="Delete target">
              &#128465;
            </button>
          </div>
        </td>
      </tr>
    `;
  }).join("");
}

function renderGuestMonitors() {
  const tbody = document.getElementById("monitors-table-body");
  const countBadge = document.getElementById("monitor-count");

  if (countBadge) {
    countBadge.textContent = "1 Target";
  }

  if (tbody) {
    tbody.innerHTML = `
      <tr>
        <td>
          <div class="service-cell">
            <span class="service-name">GoodAI Monitor Health</span>
            <span class="service-tags">Internal Telemetry</span>
          </div>
        </td>
        <td><code class="url-badge">GET /healthz</code></td>
        <td>5 seconds</td>
        <td>--</td>
        <td>100.0%</td>
        <td><span class="badge badge-up">UP</span></td>
        <td style="text-align: right;">
          <span style="font-size: 0.75rem; color: var(--text-muted);">System</span>
        </td>
      </tr>
    `;
  }
}

function updateQuotaBadge(currentCount) {
  const quotaEl = document.getElementById("monitor-quota-display");
  const btnAdd = document.getElementById("btn-open-add-monitor");

  if (quotaEl) {
    quotaEl.textContent = `${currentCount} / 10 Targets`;
  }

  if (btnAdd) {
    if (currentCount >= 10) {
      btnAdd.disabled = true;
      btnAdd.title = "Quota limit reached (maximum 10 active monitors)";
    } else {
      btnAdd.disabled = false;
      btnAdd.title = "Add a new monitoring target";
    }
  }
}

function getBadgeClassForStatus(status) {
  switch (status) {
    case "UP": return "badge-up";
    case "DOWN": return "badge-down";
    case "DEGRADED": return "badge-degraded";
    case "PAUSED": return "badge-paused";
    case "PENDING":
    default: return "badge-pending";
  }
}

// Global actions for inline button clicks
window.togglePauseMonitor = async function(id) {
  try {
    const response = await fetch(`/api/v1/monitors/${id}/pause`, {
      method: "POST"
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || "Failed to toggle pause");
    }

    const action = data.monitor.status === "PAUSED" ? "paused" : "resumed";
    showToast(`Monitor ${action} successfully.`, "info");
    loadMonitors();
  } catch (err) {
    showToast(err.message, "error");
  }
};

window.openEditMonitorModal = async function(id) {
  try {
    const response = await fetch(`/api/v1/monitors/${id}`, { cache: "no-store" });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || "Failed to load monitor details");
    }

    const m = data.monitor;
    document.getElementById("edit-monitor-id").value = m.id;
    document.getElementById("edit-monitor-name").value = m.name;
    document.getElementById("edit-monitor-url").value = m.url;
    document.getElementById("edit-monitor-interval").value = m.interval_seconds;
    document.getElementById("edit-monitor-timeout").value = m.timeout_seconds;
    document.getElementById("edit-monitor-status-code").value = m.expected_status_code;
    document.getElementById("edit-monitor-keyword").value = m.keyword || "";

    const editModal = document.getElementById("edit-monitor-modal");
    const editAlert = document.getElementById("edit-monitor-alert");
    clearAlert(editAlert);
    openModal(editModal);
  } catch (err) {
    showToast(err.message, "error");
  }
};

window.openDeleteMonitorModal = function(id, name) {
  document.getElementById("delete-monitor-id").value = id;
  document.getElementById("delete-monitor-name-label").textContent = name;
  const deleteModal = document.getElementById("delete-monitor-modal");
  openModal(deleteModal);
};

function startMonitorPolling() {
  stopMonitorPolling();
  monitorPollInterval = setInterval(loadMonitors, 8000);
}

function stopMonitorPolling() {
  if (monitorPollInterval) {
    clearInterval(monitorPollInterval);
    monitorPollInterval = null;
  }
}

// Toast Notifications
function showToast(message, type = "info") {
  const container = document.getElementById("toast-container");
  if (!container) return;

  const toast = document.createElement("div");
  toast.className = `toast toast-${type}`;
  toast.textContent = message;

  container.appendChild(toast);

  setTimeout(() => {
    toast.style.transition = "opacity 0.25s ease, transform 0.25s ease";
    toast.style.opacity = "0";
    toast.style.transform = "translateY(8px)";
    setTimeout(() => {
      if (toast.parentNode) toast.parentNode.removeChild(toast);
    }, 250);
  }, 3500);
}

// System Health Polling
async function pollHealth() {
  const lastPollEl = document.getElementById("last-poll-time");
  try {
    const response = await fetch("/healthz", { cache: "no-store" });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const data = await response.json();
    updateUIWithHealthData(data);
    if (lastPollEl) lastPollEl.textContent = new Date().toLocaleTimeString("en-NZ", { hour12: false });
  } catch (error) {
    console.warn("Failed to poll /healthz:", error);
    markSystemDegraded("Failed to connect to health endpoint");
    if (lastPollEl) lastPollEl.textContent = "Connection Error";
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

function openModal(modal) {
  if (!modal) return;
  modal.style.display = "flex";
  const firstInput = modal.querySelector("input:not([type=hidden])");
  if (firstInput) {
    setTimeout(() => firstInput.focus(), 50);
  }
}

function closeModal(modal) {
  if (!modal) return;
  modal.style.display = "none";
  const alert = modal.querySelector(".form-alert");
  if (alert) clearAlert(alert);
}

function showAlert(el, msg, type) {
  if (!el) return;
  el.textContent = msg;
  el.className = `form-alert alert-${type}`;
  el.style.display = "block";
}

function clearAlert(el) {
  if (!el) return;
  el.textContent = "";
  el.style.display = "none";
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

function truncateText(str, maxLen) {
  if (!str) return "";
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen - 3) + "...";
}

function escapeHTML(str) {
  if (!str) return "";
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
