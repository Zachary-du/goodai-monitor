// GoodAI Monitor - Web Dashboard Application Logic (100% English)

let currentUser = null;
let monitorPollInterval = null;
let cachedMonitors = [];

document.addEventListener("DOMContentLoaded", () => {
  initTheme();
  initClock();
  initAuth();
  initMonitors();
  initIncidents();
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

  // Hero CTA Triggers
  document.querySelectorAll(".btn-launch-free").forEach((btn) => {
    btn.addEventListener("click", () => openModal(registerModal));
  });

  const btnExplore = document.getElementById("btn-explore-telemetry");
  if (btnExplore) {
    btnExplore.addEventListener("click", () => {
      const target = document.getElementById("hero-showcase") || document.getElementById("live-status-section");
      if (target) target.scrollIntoView({ behavior: "smooth" });
    });
  }

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

    const landingView = document.getElementById("landing-view");
    const dashboardView = document.getElementById("dashboard-view");
    if (landingView) landingView.style.display = "none";
    if (dashboardView) dashboardView.style.display = "flex";

    const displayName = user.name || user.email.split("@")[0];
    userDisplayName.textContent = displayName;
    userAvatar.textContent = (displayName.charAt(0) || "U").toUpperCase();

    // Show authenticated monitor controls
    const btnOpenAdd = document.getElementById("btn-open-add-monitor");
    const quotaDisplay = document.getElementById("monitor-quota-display");
    if (btnOpenAdd) btnOpenAdd.style.display = "inline-flex";
    if (quotaDisplay) quotaDisplay.style.display = "inline-block";

    loadMonitors();
    loadIncidents();
    startMonitorPolling();
  }

  function setGuestState() {
    currentUser = null;
    cachedMonitors = [];
    guestNav.style.display = "flex";
    userNav.style.display = "none";

    const landingView = document.getElementById("landing-view");
    const dashboardView = document.getElementById("dashboard-view");
    if (landingView) landingView.style.display = "flex";
    if (dashboardView) dashboardView.style.display = "none";

    // Hide authenticated monitor controls
    const btnOpenAdd = document.getElementById("btn-open-add-monitor");
    const quotaDisplay = document.getElementById("monitor-quota-display");
    if (btnOpenAdd) btnOpenAdd.style.display = "none";
    if (quotaDisplay) quotaDisplay.style.display = "none";

    stopMonitorPolling();
    renderGuestMonitors();
    renderGuestIncidents();
  }
}

// Monitor Target Management
function initMonitors() {
  const addModal = document.getElementById("add-monitor-modal");
  const editModal = document.getElementById("edit-monitor-modal");
  const deleteModal = document.getElementById("delete-monitor-modal");
  const detailsModal = document.getElementById("monitor-details-modal");

  const btnOpenAdd = document.getElementById("btn-open-add-monitor");
  const btnCloseAdd = document.getElementById("btn-close-add-monitor");
  const btnCloseEdit = document.getElementById("btn-close-edit-monitor");
  const btnCloseDelete = document.getElementById("btn-close-delete-modal");
  const btnCloseDetails = document.getElementById("btn-close-details-modal");
  const btnCancelDelete = document.getElementById("btn-cancel-delete");
  const btnConfirmDelete = document.getElementById("btn-confirm-delete");

  const addForm = document.getElementById("add-monitor-form");
  const editForm = document.getElementById("edit-monitor-form");
  const addAlert = document.getElementById("add-monitor-alert");
  const editAlert = document.getElementById("edit-monitor-alert");
  const addSubmitBtn = document.getElementById("add-monitor-submit-btn");
  const editSubmitBtn = document.getElementById("edit-monitor-submit-btn");

  // Modal backdrop close listeners
  [addModal, editModal, deleteModal, detailsModal].forEach((modal) => {
    if (modal) {
      modal.addEventListener("click", (e) => {
        if (e.target === modal) {
          if (modal === detailsModal) {
            closeMonitorDetailsModal();
          } else {
            closeModal(modal);
          }
        }
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
  if (btnCloseDetails) btnCloseDetails.addEventListener("click", () => closeMonitorDetailsModal());
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
    cachedMonitors = data.monitors || [];
    renderMonitorsTable(cachedMonitors);
    updateQuotaBadge(cachedMonitors.length);
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
          <div class="service-cell" onclick="openMonitorDetailsModal(${m.id})" style="cursor: pointer;" title="Click to view telemetry details">
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
            <button class="btn-action" onclick="openMonitorDetailsModal(${m.id})" title="View telemetry and latency history">
              &#128200; Details
            </button>
            <button class="btn-action btn-action-pause" onclick="togglePauseMonitor(${m.id})" title="${pauseTitle}">
              ${pauseLabel}
            </button>
            <button class="btn-action" onclick="openEditMonitorModal(${m.id})" title="Edit target settings">
              &#9998; Edit
            </button>
            <button class="btn-action btn-action-delete" onclick="openDeleteMonitorModal(${m.id})" title="Delete target">
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

window.openDeleteMonitorModal = function(id) {
  const m = cachedMonitors.find(x => x.id === id);
  const name = m ? m.name : "";
  document.getElementById("delete-monitor-id").value = id;
  document.getElementById("delete-monitor-name-label").textContent = name;
  const deleteModal = document.getElementById("delete-monitor-modal");
  openModal(deleteModal);
};

// Monitor Details View & Telemetry Management
let currentDetailsMonitorID = null;
let detailsPollInterval = null;

window.openMonitorDetailsModal = async function(id) {
  currentDetailsMonitorID = id;
  const detailsModal = document.getElementById("monitor-details-modal");
  if (!detailsModal) return;

  const m = cachedMonitors.find(x => x.id === id);
  if (m) {
    document.getElementById("details-monitor-name").textContent = m.name;
    document.getElementById("details-monitor-url").textContent = m.url;
    const badge = document.getElementById("details-monitor-status-badge");
    badge.className = `badge ${getBadgeClassForStatus(m.status)}`;
    badge.textContent = m.status;
  }

  openModal(detailsModal);
  await loadMonitorTelemetry(id);

  // Live auto-refresh telemetry polling every 10 seconds while modal is open
  stopDetailsPolling();
  detailsPollInterval = setInterval(() => {
    if (currentDetailsMonitorID) {
      loadMonitorTelemetry(currentDetailsMonitorID);
    }
  }, 10000);
};

window.closeMonitorDetailsModal = function() {
  stopDetailsPolling();
  currentDetailsMonitorID = null;
  const detailsModal = document.getElementById("monitor-details-modal");
  if (detailsModal) closeModal(detailsModal);
};

function stopDetailsPolling() {
  if (detailsPollInterval) {
    clearInterval(detailsPollInterval);
    detailsPollInterval = null;
  }
}

async function loadMonitorTelemetry(id) {
  try {
    const [metricsRes, checksRes] = await Promise.all([
      fetch(`/api/v1/monitors/${id}/metrics`, { cache: "no-store" }),
      fetch(`/api/v1/monitors/${id}/checks?limit=50`, { cache: "no-store" })
    ]);

    if (metricsRes.ok) {
      const metrics = await metricsRes.json();
      updateDetailsMetrics(metrics);
    }

    if (checksRes.ok) {
      const checksData = await checksRes.json();
      renderDetailsChecksTable(checksData.checks || []);
    }
  } catch (err) {
    console.warn("Failed to load monitor telemetry:", err);
  }
}

function updateDetailsMetrics(metrics) {
  const u24 = document.getElementById("details-uptime-24h");
  const u7 = document.getElementById("details-uptime-7d");
  const u30 = document.getElementById("details-uptime-30d");
  const avgLat = document.getElementById("details-avg-latency");
  const checksCount = document.getElementById("details-checks-count-24h");

  if (u24 && typeof metrics.uptime_24h === "number") {
    u24.textContent = `${metrics.uptime_24h.toFixed(1)}%`;
    u24.style.color = metrics.uptime_24h >= 99 ? "var(--color-up)" : "var(--color-degraded)";
  }
  if (u7 && typeof metrics.uptime_7d === "number") {
    u7.textContent = `${metrics.uptime_7d.toFixed(1)}%`;
    u7.style.color = metrics.uptime_7d >= 99 ? "var(--color-up)" : "var(--color-degraded)";
  }
  if (u30 && typeof metrics.uptime_30d === "number") {
    u30.textContent = `${metrics.uptime_30d.toFixed(1)}%`;
    u30.style.color = metrics.uptime_30d >= 99 ? "var(--color-up)" : "var(--color-degraded)";
  }
  if (avgLat) {
    avgLat.textContent = metrics.avg_latency_24h > 0 ? `${metrics.avg_latency_24h.toFixed(1)} ms` : "-- ms";
  }
  if (checksCount && typeof metrics.total_checks_24h === "number") {
    checksCount.textContent = `${metrics.total_checks_24h} probes in 24h`;
  }

  renderLatencySparkline(metrics.latency_history || []);
}

function renderLatencySparkline(points) {
  const svg = document.getElementById("details-latency-chart");
  if (!svg) return;

  if (!points || points.length === 0) {
    svg.innerHTML = `
      <text x="380" y="85" text-anchor="middle" fill="var(--text-muted)" font-size="13">
        No check telemetry recorded yet. Background prober will populate live samples.
      </text>
    `;
    return;
  }

  const width = 760;
  const height = 160;
  const padX = 45;
  const padY = 25;
  const innerW = width - padX * 2;
  const innerH = height - padY * 2;

  let latencies = points.map(p => p.latency_ms);
  let minLat = Math.min(...latencies);
  let maxLat = Math.max(...latencies);
  if (maxLat === minLat) {
    maxLat = minLat + 10;
  }

  const scaleX = (i) => padX + (i / Math.max(points.length - 1, 1)) * innerW;
  const scaleY = (val) => padY + innerH - ((val - minLat) / (maxLat - minLat)) * innerH;

  const coordinates = points.map((p, i) => ({
    x: scaleX(i),
    y: scaleY(p.latency_ms),
    point: p
  }));

  const pathD = coordinates.reduce((acc, c, i) => `${acc} ${i === 0 ? 'M' : 'L'} ${c.x.toFixed(1)} ${c.y.toFixed(1)}`, "");
  const areaD = `${pathD} L ${coordinates[coordinates.length - 1].x.toFixed(1)} ${height - padY} L ${coordinates[0].x.toFixed(1)} ${height - padY} Z`;

  const midLat = Math.round((minLat + maxLat) / 2);
  const yMax = scaleY(maxLat);
  const yMid = scaleY(midLat);
  const yMin = scaleY(minLat);

  let dotsSvg = "";
  if (points.length <= 40) {
    dotsSvg = coordinates.map(c => `
      <circle cx="${c.x.toFixed(1)}" cy="${c.y.toFixed(1)}" r="3" fill="${c.point.status === 'UP' ? 'var(--color-up)' : 'var(--color-down)'}">
        <title>${c.point.latency_ms} ms (${c.point.status})</title>
      </circle>
    `).join("");
  }

  svg.innerHTML = `
    <defs>
      <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#2563eb" stop-opacity="0.35"/>
        <stop offset="100%" stop-color="#2563eb" stop-opacity="0.0"/>
      </linearGradient>
    </defs>
    <!-- Grid Lines & Latency Labels -->
    <line x1="${padX}" y1="${yMax.toFixed(1)}" x2="${width - padX}" y2="${yMax.toFixed(1)}" stroke="var(--border-color)" stroke-dasharray="3,3"/>
    <text x="${padX - 8}" y="${(yMax + 4).toFixed(1)}" text-anchor="end" fill="var(--text-muted)" font-size="10">${maxLat}ms</text>

    <line x1="${padX}" y1="${yMid.toFixed(1)}" x2="${width - padX}" y2="${yMid.toFixed(1)}" stroke="var(--border-color)" stroke-dasharray="3,3"/>
    <text x="${padX - 8}" y="${(yMid + 4).toFixed(1)}" text-anchor="end" fill="var(--text-muted)" font-size="10">${midLat}ms</text>

    <line x1="${padX}" y1="${yMin.toFixed(1)}" x2="${width - padX}" y2="${yMin.toFixed(1)}" stroke="var(--border-color)" stroke-dasharray="3,3"/>
    <text x="${padX - 8}" y="${(yMin + 4).toFixed(1)}" text-anchor="end" fill="var(--text-muted)" font-size="10">${minLat}ms</text>

    <!-- Gradient Fill Area -->
    <path d="${areaD}" fill="url(#latencyGradient)"/>

    <!-- Latency Curve -->
    <path d="${pathD}" fill="none" stroke="#2563eb" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>

    <!-- Interactive Data Points -->
    ${dotsSvg}
  `;
}

function renderDetailsChecksTable(checks) {
  const tbody = document.getElementById("details-checks-table-body");
  const countLabel = document.getElementById("details-checks-table-count");
  if (!tbody) return;

  if (countLabel) {
    countLabel.textContent = `Showing ${checks.length} recent probes`;
  }

  if (checks.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="6" style="text-align: center; color: var(--text-muted); padding: 2rem;">
          No check executions recorded yet.
        </td>
      </tr>
    `;
    return;
  }

  tbody.innerHTML = checks.map(c => {
    const badgeClass = getBadgeClassForStatus(c.status);
    const timeStr = formatTimestamp(c.created_at);
    const codeDisplay = c.status_code > 0 ? c.status_code : "--";
    const codeColor = c.status_code >= 200 && c.status_code < 400 ? "var(--color-up)" : "var(--color-down)";

    let diagText = `<span style="color: var(--color-up);">&#10003; Healthy</span>`;
    if (c.error_message) {
      diagText = `<span style="color: var(--color-down); font-weight: 500;">${escapeHTML(c.error_message)}</span>`;
    } else if (c.cert_expiry_days !== undefined && c.cert_expiry_days !== null) {
      diagText = `<span style="color: var(--text-secondary);">Cert: ${c.cert_expiry_days}d remaining</span>`;
    }

    const timings = `DNS: ${c.dns_lookup_ms}ms &bull; TCP: ${c.tcp_connect_ms}ms &bull; TLS: ${c.tls_handshake_ms}ms &bull; TTFB: ${c.ttfb_ms}ms`;

    return `
      <tr>
        <td style="font-family: var(--font-mono); font-size: 0.75rem; white-space: nowrap;">${timeStr}</td>
        <td><span class="badge ${badgeClass}" style="font-size: 0.6875rem; padding: 0.15rem 0.45rem;">${c.status}</span></td>
        <td><span style="font-family: var(--font-mono); font-weight: 600; color: ${codeColor};">${codeDisplay}</span></td>
        <td><span style="font-family: var(--font-mono); font-weight: 600;">${c.latency_ms} ms</span></td>
        <td style="font-size: 0.75rem; color: var(--text-muted); font-family: var(--font-mono);">${timings}</td>
        <td style="font-size: 0.75rem;">${diagText}</td>
      </tr>
    `;
  }).join("");
}

function formatTimestamp(isoStr) {
  if (!isoStr) return "--";
  try {
    const d = new Date(isoStr);
    return d.toLocaleString("en-NZ", {
      timeZone: "Pacific/Auckland",
      month: "short",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false
    });
  } catch (e) {
    return isoStr;
  }
}

function startMonitorPolling() {
  stopMonitorPolling();
  monitorPollInterval = setInterval(() => {
    loadMonitors();
    loadIncidents();
  }, 8000);
}

function stopMonitorPolling() {
  if (monitorPollInterval) {
    clearInterval(monitorPollInterval);
    monitorPollInterval = null;
  }
}

// Incident Hub Management (AC-8)
let currentIncidentFilter = "all";
let cachedIncidents = [];

function initIncidents() {
  const filterAll = document.getElementById("filter-incident-all");
  const filterOpen = document.getElementById("filter-incident-open");
  const filterResolved = document.getElementById("filter-incident-resolved");
  const btnScroll = document.getElementById("btn-scroll-to-incidents");

  if (filterAll) {
    filterAll.addEventListener("click", () => setIncidentFilter("all"));
  }
  if (filterOpen) {
    filterOpen.addEventListener("click", () => setIncidentFilter("open"));
  }
  if (filterResolved) {
    filterResolved.addEventListener("click", () => setIncidentFilter("resolved"));
  }
  if (btnScroll) {
    btnScroll.addEventListener("click", () => {
      const section = document.getElementById("incidents-section");
      if (section) {
        section.scrollIntoView({ behavior: "smooth" });
      }
    });
  }
}

function setIncidentFilter(filter) {
  currentIncidentFilter = filter;
  ["all", "open", "resolved"].forEach(f => {
    const el = document.getElementById(`filter-incident-${f}`);
    if (el) {
      if (f === filter) {
        el.classList.add("active");
      } else {
        el.classList.remove("active");
      }
    }
  });
  loadIncidents();
}

async function loadIncidents() {
  if (!currentUser) return;

  try {
    // 1. Fetch incident summary to update active outage banner
    const summaryRes = await fetch("/api/v1/incidents/summary", { cache: "no-store" });
    if (summaryRes.ok) {
      const summary = await summaryRes.json();
      updateOutageBanner(summary.open_count);
    }

    // 2. Fetch incidents with current filter
    const incRes = await fetch(`/api/v1/incidents?status=${currentIncidentFilter}&limit=50`, { cache: "no-store" });
    if (!incRes.ok) return;

    const data = await incRes.json();
    cachedIncidents = data.incidents || [];
    renderIncidentsTable(cachedIncidents);
  } catch (err) {
    console.warn("Failed to load incidents:", err);
  }
}

function updateOutageBanner(openCount) {
  const banner = document.getElementById("active-outage-banner");
  const bannerTitle = document.getElementById("outage-banner-title");
  const bannerMessage = document.getElementById("outage-banner-message");

  const sysBannerBadge = document.getElementById("banner-badge");
  const sysBannerText = document.getElementById("banner-status-text");
  const sysBannerHeading = document.getElementById("banner-heading");
  const sysBannerSubtext = document.getElementById("banner-subtext");
  const brandIndicator = document.getElementById("brand-indicator");

  if (openCount > 0) {
    if (banner) banner.style.display = "flex";
    if (bannerTitle) bannerTitle.textContent = `${openCount} Active Outage${openCount === 1 ? "" : "s"} Confirmed`;
    if (bannerMessage) bannerMessage.textContent = `${openCount} monitored target${openCount === 1 ? " is" : "s are"} currently down after consecutive failed checks. Alert emails dispatched.`;

    // Elevate overall system banner to DOWN
    if (sysBannerBadge) sysBannerBadge.className = "banner-status-badge badge-down";
    if (sysBannerText) sysBannerText.textContent = "OUTAGE DETECTED";
    if (sysBannerHeading) sysBannerHeading.textContent = `${openCount} Service Outage${openCount === 1 ? "" : "s"} In Progress`;
    if (sysBannerSubtext) sysBannerSubtext.textContent = "Automated probes detected downtime. Investigating root cause.";
    if (brandIndicator) brandIndicator.className = "status-indicator-dot offline";
  } else {
    if (banner) banner.style.display = "none";
  }
}

function renderIncidentsTable(incidents) {
  const emptyState = document.getElementById("incidents-empty-state");
  const tableWrapper = document.getElementById("incidents-table-wrapper");
  const tbody = document.getElementById("incidents-table-body");
  const countBadge = document.getElementById("incident-count-display");

  if (countBadge) {
    countBadge.textContent = `${incidents.length} Recorded`;
  }

  if (incidents.length === 0) {
    if (emptyState) emptyState.style.display = "flex";
    if (tableWrapper) tableWrapper.style.display = "none";
    return;
  }

  if (emptyState) emptyState.style.display = "none";
  if (tableWrapper) tableWrapper.style.display = "block";

  if (!tbody) return;

  tbody.innerHTML = incidents.map(inc => {
    const isResolved = inc.status === "RESOLVED";
    const badgeClass = isResolved ? "badge-up" : "badge-down";
    const statusText = inc.status;
    const durationText = isResolved && inc.duration_seconds > 0 
      ? formatDurationSeconds(inc.duration_seconds)
      : `<span style="color: var(--color-down); font-weight: 600;">Ongoing</span>`;

    return `
      <tr>
        <td>
          <div class="service-cell">
            <span class="service-name">${escapeHTML(inc.monitor_name || "Monitor #" + inc.monitor_id)}</span>
            <span class="service-tags">${escapeHTML(inc.monitor_url || "")}</span>
          </div>
        </td>
        <td><span class="badge ${badgeClass}">${statusText}</span></td>
        <td><span style="font-family: var(--font-mono); font-size: 0.8125rem;">${formatTimestamp(inc.started_at)}</span></td>
        <td><span style="font-family: var(--font-mono); font-size: 0.8125rem;">${durationText}</span></td>
        <td><span style="font-size: 0.8125rem; color: var(--text-secondary); max-width: 320px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHTML(inc.cause)}">${escapeHTML(inc.cause || "Probe failure")}</span></td>
      </tr>
    `;
  }).join("");
}

function renderGuestIncidents() {
  const banner = document.getElementById("active-outage-banner");
  if (banner) banner.style.display = "none";

  const emptyState = document.getElementById("incidents-empty-state");
  const tableWrapper = document.getElementById("incidents-table-wrapper");
  const countBadge = document.getElementById("incident-count-display");

  if (emptyState) emptyState.style.display = "flex";
  if (tableWrapper) tableWrapper.style.display = "none";
  if (countBadge) countBadge.textContent = "0 Recorded";
}

function formatDurationSeconds(seconds) {
  if (seconds <= 0) return "0s";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
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

  // Update Landing Page Live Telemetry Proof widgets
  const landingUptime = document.getElementById("landing-uptime-val");
  const landingRss = document.getElementById("landing-rss-val");
  const landingGoro = document.getElementById("landing-goroutines-val");
  const landingBadge = document.getElementById("landing-telemetry-badge");

  if (landingUptime && typeof data.uptime_seconds === "number") {
    landingUptime.textContent = formatUptime(data.uptime_seconds);
  }
  if (landingRss && typeof data.memory_rss_bytes === "number") {
    const mib = data.memory_rss_bytes / (1024 * 1024);
    landingRss.textContent = `${mib.toFixed(1)} MiB`;
  }
  if (landingGoro && typeof data.go_goroutine_count === "number") {
    landingGoro.textContent = `Active Goroutines: ${data.go_goroutine_count}`;
  }
  if (landingBadge) {
    if (data.status === "ok" && data.database === "connected") {
      landingBadge.textContent = "100% OPERATIONAL";
      if (landingBadge.parentElement) landingBadge.parentElement.className = "banner-status-badge badge-up";
    } else {
      landingBadge.textContent = "DEGRADED";
      if (landingBadge.parentElement) landingBadge.parentElement.className = "banner-status-badge badge-degraded";
    }
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
