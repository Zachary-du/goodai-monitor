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
  initSecurityAudit();
  initShowcaseTabs();
  initPublicAudit();
  initFAQ();
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

  // Check URL parameters for OAuth redirect error
  const urlParams = new URLSearchParams(window.location.search);
  if (urlParams.has("error")) {
    const errorMsg = urlParams.get("error");
    showToast(`Authentication notice: ${errorMsg}`, "error");
    window.history.replaceState({}, document.title, window.location.pathname);
  }

  // Feature detection for Google SSO
  let googleSSOEnabled = true;
  async function checkAuthProviders() {
    try {
      const resp = await fetch("/api/v1/auth/providers");
      if (resp.ok) {
        const data = await resp.json();
        googleSSOEnabled = !!data.google;
        const googleBtns = document.querySelectorAll(".btn-google-sso");
        googleBtns.forEach((btn) => {
          if (!googleSSOEnabled) {
            btn.setAttribute("title", "Google SSO is not configured on this server");
            btn.classList.add("disabled");
          } else {
            btn.removeAttribute("title");
            btn.classList.remove("disabled");
          }
        });
      }
    } catch (e) {
      console.warn("Failed to check auth providers:", e);
    }
  }
  checkAuthProviders();

  // Attach click handler for Google SSO buttons
  document.querySelectorAll(".btn-google-sso").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      if (!googleSSOEnabled) {
        e.preventDefault();
        showToast("Google Single Sign-On is not configured on this server.", "warning");
      }
    });
  });

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
    loadLatestSecurityAuditReport();
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
    updateFleetMetrics([]);
    renderGuestMonitors();
    renderGuestIncidents();
    resetSecurityAuditState();
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
    populateAuditTargetDropdown(cachedMonitors);
    updateFleetMetrics(cachedMonitors);
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
    countBadge.textContent = "0 Targets";
  }

  if (tbody) {
    tbody.innerHTML = `
      <tr>
        <td colspan="7" style="text-align: center; padding: 2.5rem 1rem;">
          <div style="color: var(--text-secondary); margin-bottom: 0.5rem; font-size: 0.9375rem; font-weight: 500;">
            Sign in to configure and monitor your web services.
          </div>
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
// Customer Fleet Telemetry Aggregation Engine
function updateFleetMetrics(monitors) {
  const slaVal = document.getElementById("fleet-sla-val");
  const slaBadge = document.getElementById("fleet-sla-badge");
  const slaSubtext = document.getElementById("fleet-sla-subtext");

  const targetsVal = document.getElementById("fleet-targets-val");
  const targetsBadge = document.getElementById("fleet-targets-badge");
  const targetsSubtext = document.getElementById("fleet-targets-subtext");

  const latencyVal = document.getElementById("fleet-latency-val");
  const latencyBadge = document.getElementById("fleet-latency-badge");
  const latencySubtext = document.getElementById("fleet-latency-subtext");

  const outagesVal = document.getElementById("fleet-outages-val");
  const outagesBadge = document.getElementById("fleet-outages-badge");
  const outagesSubtext = document.getElementById("fleet-outages-subtext");

  const bannerBadge = document.getElementById("banner-badge");
  const bannerText = document.getElementById("banner-status-text");
  const bannerHeading = document.getElementById("banner-heading");
  const bannerSubtext = document.getElementById("banner-subtext");
  const brandIndicator = document.getElementById("brand-indicator");
  const lastPollEl = document.getElementById("last-poll-time");

  if (lastPollEl) {
    lastPollEl.textContent = new Date().toLocaleTimeString("en-NZ", { hour12: false });
  }

  if (!monitors || monitors.length === 0) {
    // Zero active monitors configured
    if (slaVal) slaVal.textContent = "--";
    if (slaBadge) {
      slaBadge.className = "badge-subtle";
      slaBadge.textContent = "STANDBY";
    }
    if (slaSubtext) slaSubtext.textContent = "No active targets configured";

    if (targetsVal) targetsVal.textContent = "0 / 10";
    if (targetsBadge) {
      targetsBadge.className = "badge-subtle badge-up";
      targetsBadge.textContent = "Available";
    }
    if (targetsSubtext) targetsSubtext.textContent = "10 probe slots available";

    if (latencyVal) latencyVal.textContent = "--";
    if (latencyBadge) {
      latencyBadge.className = "badge-subtle";
      latencyBadge.textContent = "--";
    }
    if (latencySubtext) latencySubtext.textContent = "Awaiting first probe";

    if (outagesVal) outagesVal.textContent = "0";
    if (outagesBadge) {
      outagesBadge.className = "badge-subtle badge-up";
      outagesBadge.textContent = "ALL OPERATIONAL";
    }
    if (outagesSubtext) outagesSubtext.textContent = "Zero active incidents";

    if (bannerBadge) bannerBadge.className = "banner-status-badge badge-pending";
    if (bannerText) bannerText.textContent = "STANDBY";
    if (bannerHeading) bannerHeading.textContent = "No Monitoring Targets Configured";
    if (bannerSubtext) bannerSubtext.textContent = "Add your first service or API endpoint to begin synthetic health probing.";
    if (brandIndicator) brandIndicator.className = "status-indicator-dot online";
    return;
  }

  // Active targets breakdown
  const activeMonitors = monitors.filter(m => m.status !== "PAUSED");
  const pausedMonitors = monitors.filter(m => m.status === "PAUSED");
  const downMonitors = monitors.filter(m => m.status === "DOWN");
  const degradedMonitors = monitors.filter(m => m.status === "DEGRADED");

  // 1. Fleet SLA Calculation (Average uptime % across active monitors)
  if (activeMonitors.length > 0) {
    const totalUptime = activeMonitors.reduce((acc, m) => acc + (typeof m.uptime_percent === "number" ? m.uptime_percent : 100), 0);
    const avgUptime = totalUptime / activeMonitors.length;
    if (slaVal) slaVal.textContent = `${avgUptime.toFixed(2)}%`;
    if (slaBadge) {
      if (downMonitors.length > 0) {
        slaBadge.className = "badge-subtle badge-down";
        slaBadge.textContent = "OUTAGE";
      } else if (avgUptime < 99.0 || degradedMonitors.length > 0) {
        slaBadge.className = "badge-subtle badge-degraded";
        slaBadge.textContent = "DEGRADED";
      } else {
        slaBadge.className = "badge-subtle badge-up";
        slaBadge.textContent = "HEALTHY";
      }
    }
    if (slaSubtext) slaSubtext.textContent = `Across ${activeMonitors.length} active target${activeMonitors.length === 1 ? '' : 's'}`;
  } else {
    if (slaVal) slaVal.textContent = "--";
    if (slaBadge) {
      slaBadge.className = "badge-subtle";
      slaBadge.textContent = "PAUSED";
    }
    if (slaSubtext) slaSubtext.textContent = "All targets currently paused";
  }

  // 2. Monitored Services Quota
  if (targetsVal) targetsVal.textContent = `${monitors.length} / 10`;
  if (targetsBadge) {
    targetsBadge.className = pausedMonitors.length > 0 ? "badge-subtle badge-paused" : "badge-subtle badge-up";
    targetsBadge.textContent = pausedMonitors.length > 0 ? `${pausedMonitors.length} Paused` : "All Active";
  }
  if (targetsSubtext) {
    const remaining = 10 - monitors.length;
    targetsSubtext.textContent = `${remaining} available slot${remaining === 1 ? '' : 's'}`;
  }

  // 3. Average Response Latency
  const responsiveMonitors = monitors.filter(m => typeof m.last_latency_ms === "number" && m.last_latency_ms > 0);
  if (responsiveMonitors.length > 0) {
    const totalLatency = responsiveMonitors.reduce((acc, m) => acc + m.last_latency_ms, 0);
    const avgLatency = Math.round(totalLatency / responsiveMonitors.length);
    if (latencyVal) latencyVal.textContent = `${avgLatency} ms`;
    if (latencyBadge) {
      if (avgLatency <= 200) {
        latencyBadge.className = "badge-subtle badge-up";
        latencyBadge.textContent = "FAST";
      } else if (avgLatency <= 500) {
        latencyBadge.className = "badge-subtle badge-degraded";
        latencyBadge.textContent = "MODERATE";
      } else {
        latencyBadge.className = "badge-subtle badge-down";
        latencyBadge.textContent = "HIGH";
      }
    }
    if (latencySubtext) latencySubtext.textContent = "Fleet synthetic TTFB";
  } else {
    if (latencyVal) latencyVal.textContent = "--";
    if (latencyBadge) {
      latencyBadge.className = "badge-subtle";
      latencyBadge.textContent = "--";
    }
    if (latencySubtext) latencySubtext.textContent = "Awaiting first response";
  }

  // 4. Active Outages
  const outageCount = downMonitors.length;
  if (outagesVal) outagesVal.textContent = `${outageCount}`;
  if (outagesBadge) {
    if (outageCount === 0) {
      outagesBadge.className = "badge-subtle badge-up";
      outagesBadge.textContent = "ALL OPERATIONAL";
    } else {
      outagesBadge.className = "badge-subtle badge-down";
      outagesBadge.textContent = `${outageCount} ACTIVE`;
    }
  }
  if (outagesSubtext) {
    outagesSubtext.textContent = outageCount === 0 ? "Zero confirmed outages" : "Immediate attention required";
  }

  // 5. System Fleet Banner
  if (outageCount > 0) {
    if (bannerBadge) bannerBadge.className = "banner-status-badge badge-down";
    if (bannerText) bannerText.textContent = "OUTAGE DETECTED";
    if (bannerHeading) bannerHeading.textContent = `Fleet Incident Detected (${outageCount} Outage${outageCount === 1 ? '' : 's'})`;
    if (bannerSubtext) bannerSubtext.textContent = "One or more monitored endpoints are currently failing synthetic health probes.";
    if (brandIndicator) brandIndicator.className = "status-indicator-dot offline";
  } else if (degradedMonitors.length > 0) {
    if (bannerBadge) bannerBadge.className = "banner-status-badge badge-degraded";
    if (bannerText) bannerText.textContent = "DEGRADED";
    if (bannerHeading) bannerHeading.textContent = "Fleet Latency Degraded";
    if (bannerSubtext) bannerSubtext.textContent = "All services reachable, but one or more targets exhibit elevated response times.";
    if (brandIndicator) brandIndicator.className = "status-indicator-dot online";
  } else {
    if (bannerBadge) bannerBadge.className = "banner-status-badge badge-up";
    if (bannerText) bannerText.textContent = "OPERATIONAL";
    if (bannerHeading) bannerHeading.textContent = "All Monitored Services Operational";
    if (bannerSubtext) bannerSubtext.textContent = `Continuous background probing active across ${monitors.length} target${monitors.length === 1 ? '' : 's'}.`;
    if (brandIndicator) brandIndicator.className = "status-indicator-dot online";
  }
}

// System Connectivity Heartbeat
async function pollHealth() {
  const lastPollEl = document.getElementById("last-poll-time");
  const brandIndicator = document.getElementById("brand-indicator");
  try {
    const response = await fetch("/healthz", { cache: "no-store" });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const data = await response.json();
    if (brandIndicator) {
      brandIndicator.className = data.status === "ok" ? "status-indicator-dot online" : "status-indicator-dot offline";
    }
    if (lastPollEl && !currentUser) {
      lastPollEl.textContent = new Date().toLocaleTimeString("en-NZ", { hour12: false });
    }
  } catch (error) {
    console.warn("Failed to poll backend connectivity:", error);
    if (brandIndicator) brandIndicator.className = "status-indicator-dot offline";
    if (lastPollEl && !currentUser) lastPollEl.textContent = "Offline";
  }
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

// ==========================================================================
// Web Security & AI Defense Audit Console Logic (AC-6)
// ==========================================================================

let activeSecurityReport = null;
let activeFindingsFilter = "ALL";

function initSecurityAudit() {
  const btnRunAudit = document.getElementById("btn-run-audit");
  const targetInput = document.getElementById("audit-target-url");
  const targetSelect = document.getElementById("audit-target-select");
  const btnExport = document.getElementById("btn-export-security-report");

  if (targetSelect) {
    targetSelect.addEventListener("change", () => {
      if (targetSelect.value) {
        targetInput.value = targetSelect.value;
      }
    });
  }

  if (btnRunAudit) {
    btnRunAudit.addEventListener("click", () => {
      const url = (targetInput.value || "").trim();
      if (!url) {
        showAuditAlert("Please enter a valid target URL (e.g. https://example.com).");
        return;
      }
      runSecurityAudit(url);
    });
  }

  if (btnExport) {
    btnExport.addEventListener("click", () => {
      if (activeSecurityReport && activeSecurityReport.id) {
        window.open(`/api/v1/security/reports/${activeSecurityReport.id}/export`, "_blank");
      }
    });
  }

  // Findings filter buttons
  const filterBtns = document.querySelectorAll(".findings-filter-btn");
  filterBtns.forEach(btn => {
    btn.addEventListener("click", () => {
      filterBtns.forEach(b => b.classList.remove("active"));
      btn.classList.add("active");
      activeFindingsFilter = btn.dataset.filter || "ALL";
      if (activeSecurityReport) {
        renderAuditFindings(activeSecurityReport.findings || [], activeFindingsFilter);
      }
    });
  });
}

function populateAuditTargetDropdown(monitors) {
  const select = document.getElementById("audit-target-select");
  if (!select) return;

  const currentVal = select.value;
  select.innerHTML = '<option value="">Choose Monitored Target...</option>';

  if (Array.isArray(monitors)) {
    monitors.forEach(m => {
      if (m.url) {
        const opt = document.createElement("option");
        opt.value = m.url;
        opt.textContent = `${m.name} (${m.url})`;
        select.appendChild(opt);
      }
    });
  }

  if (currentVal) {
    select.value = currentVal;
  }
}

async function runSecurityAudit(targetURL) {
  const btnRunAudit = document.getElementById("btn-run-audit");
  const spinner = document.getElementById("audit-btn-spinner");
  const btnText = document.getElementById("audit-btn-text");

  clearAuditAlert();

  if (btnRunAudit) btnRunAudit.disabled = true;
  if (spinner) spinner.style.display = "inline-block";
  if (btnText) btnText.textContent = "Auditing 24 Security Vectors...";

  try {
    const res = await fetch("/api/v1/security/audit", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ target_url: targetURL })
    });

    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || `Security audit failed with status ${res.status}`);
    }

    const report = await res.json();
    renderSecurityAuditReport(report);
    showToast("Security & AI Defense Audit completed successfully!", "success");
  } catch (err) {
    console.error("Audit error:", err);
    showAuditAlert(err.message || "Failed to execute security audit. Please ensure the target URL is valid and publicly reachable.");
  } finally {
    if (btnRunAudit) btnRunAudit.disabled = false;
    if (spinner) spinner.style.display = "none";
    if (btnText) btnText.textContent = "▶ Run Security Audit";
  }
}

async function loadLatestSecurityAuditReport() {
  try {
    const res = await fetch("/api/v1/security/reports?limit=1", { cache: "no-store" });
    if (!res.ok) return;
    const data = await res.json();
    if (data.reports && data.reports.length > 0) {
      renderSecurityAuditReport(data.reports[0]);
    }
  } catch (err) {
    console.warn("Failed to load previous security report:", err);
  }
}

function renderSecurityAuditReport(report) {
  if (!report) return;
  activeSecurityReport = report;

  const emptyState = document.getElementById("audit-empty-state");
  const resultsContainer = document.getElementById("audit-results-container");
  const btnExport = document.getElementById("btn-export-security-report");

  if (emptyState) emptyState.style.display = "none";
  if (resultsContainer) resultsContainer.style.display = "flex";
  if (btnExport) btnExport.style.display = "inline-flex";

  // Score Wheel Animation
  const scoreEl = document.getElementById("audit-score-number");
  const wheelProgress = document.getElementById("score-wheel-progress");
  const gradePill = document.getElementById("audit-grade-pill");
  const gradeLabel = document.getElementById("audit-grade-label");

  const score = report.composite_score || 0;
  if (scoreEl) scoreEl.textContent = score;

  // Circumference: 2 * Math.PI * 68 ≈ 427.26
  const circumference = 427.26;
  const offset = circumference - (score / 100) * circumference;
  if (wheelProgress) {
    wheelProgress.style.strokeDashoffset = offset;
    if (score >= 85) {
      wheelProgress.style.stroke = "#10b981";
    } else if (score >= 70) {
      wheelProgress.style.stroke = "#06b6d4";
    } else if (score >= 50) {
      wheelProgress.style.stroke = "#f59e0b";
    } else {
      wheelProgress.style.stroke = "#ef4444";
    }
  }

  // Grade pill styling
  if (gradePill) {
    const grade = report.grade || "F";
    gradePill.textContent = `GRADE ${grade}`;
    gradePill.className = "audit-grade-pill";
    if (grade.startsWith("A")) gradePill.classList.add("grade-pill-a");
    else if (grade === "B") gradePill.classList.add("grade-pill-b");
    else if (grade === "C") gradePill.classList.add("grade-pill-c");
    else gradePill.classList.add("grade-pill-f");
  }

  if (gradeLabel) gradeLabel.textContent = report.grade_label || "Evaluated";

  // Metadata
  const metaHost = document.getElementById("audit-meta-host");
  const metaIP = document.getElementById("audit-meta-ip");
  const metaTimestamp = document.getElementById("audit-meta-timestamp");

  if (metaHost) metaHost.textContent = report.target_host || "--";
  if (metaIP) metaIP.textContent = report.target_ip || "--";
  if (metaTimestamp) metaTimestamp.textContent = report.created_at ? new Date(report.created_at).toLocaleString("en-NZ") : "Just now";

  // Render Domain Breakdown Cards
  renderDomainScores(report.domain_scores || []);

  // Update Filter Counts & Findings Table
  updateFindingsFilterCounts(report.findings || []);
  renderAuditFindings(report.findings || [], activeFindingsFilter);
}

function renderDomainScores(domainScores) {
  domainScores.forEach(d => {
    let key = d.domain_key;
    if (!key) {
      const name = (d.domain_name || "").toLowerCase();
      if (name.includes("ai")) key = "ai";
      else if (name.includes("header")) key = "headers";
      else if (name.includes("tls") || name.includes("transport")) key = "tls";
      else key = "exposure";
    } else {
      key = key.replace("ai_governance", "ai");
    }

    const pillEl = document.getElementById(`domain-pill-${key}`);
    const fillEl = document.getElementById(`domain-fill-${key}`);
    const noteEl = document.getElementById(`domain-note-${key}`);

    if (pillEl) pillEl.textContent = `${d.score} / ${d.max_score || 25} pts`;
    if (fillEl) {
      const max = d.max_score || 25;
      const pct = Math.min(100, Math.round((d.score / max) * 100));
      fillEl.style.width = `${pct}%`;
    }
    if (noteEl) noteEl.textContent = d.summary || `${d.status} posture`;
  });
}

function updateFindingsFilterCounts(findings) {
  let all = findings.length;
  let critical = 0, high = 0, medium = 0, low = 0, pass = 0;

  findings.forEach(f => {
    switch (f.severity) {
      case "CRITICAL": critical++; break;
      case "HIGH": high++; break;
      case "MEDIUM": medium++; break;
      case "LOW": low++; break;
      case "PASS": pass++; break;
    }
  });

  const setCnt = (id, val) => {
    const el = document.getElementById(id);
    if (el) el.textContent = val;
  };

  setCnt("count-all", all);
  setCnt("count-critical", critical);
  setCnt("count-high", high);
  setCnt("count-medium", medium);
  setCnt("count-low", low);
  setCnt("count-pass", pass);

  const totalEl = document.getElementById("findings-total-count");
  if (totalEl) totalEl.textContent = `${all} Checks`;
}

function renderAuditFindings(findings, filter) {
  const tbody = document.getElementById("audit-findings-body");
  if (!tbody) return;

  const filtered = findings.filter(f => {
    if (filter === "ALL") return true;
    return f.severity === filter;
  });

  if (filtered.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="5" style="text-align: center; color: var(--text-muted); padding: 2rem;">
          No findings match the selected filter "${escapeHTML(filter)}".
        </td>
      </tr>
    `;
    return;
  }

  tbody.innerHTML = filtered.map((f, idx) => {
    const hasRemediation = f.remediation && f.severity !== "PASS";
    const drawerId = `remediation-drawer-${idx}`;

    let actionCol = `<span class="badge badge-PASS" style="font-size: 0.6875rem;">Compliant</span>`;
    if (hasRemediation) {
      actionCol = `
        <button class="btn-copy-remediation" onclick="toggleRemediationDrawer('${drawerId}')">
          Remediation &darr;
        </button>
      `;
    }

    let domainName = f.domain;
    if (domainName === "ai_governance") domainName = "AI Governance";
    else if (domainName === "headers") domainName = "Security Headers";
    else if (domainName === "tls") domainName = "TLS & Transport";
    else if (domainName === "exposure") domainName = "Surface Exposure";

    let rowHTML = `
      <tr class="finding-row">
        <td><span class="badge badge-${escapeHTML(f.severity)}">${escapeHTML(f.severity)}</span></td>
        <td style="color: var(--text-secondary); font-size: 0.75rem;">${escapeHTML(domainName)}</td>
        <td><strong>${escapeHTML(f.check_name)}</strong></td>
        <td>
          <div>${escapeHTML(f.observed_value)}</div>
          ${f.recommendation ? `<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.25rem;">Recommendation: ${escapeHTML(f.recommendation)}</div>` : ""}
        </td>
        <td style="text-align: right;">${actionCol}</td>
      </tr>
    `;

    if (hasRemediation) {
      rowHTML += `
        <tr id="${drawerId}" style="display: none;">
          <td colspan="5" style="padding: 0; background: transparent;">
            <div class="remediation-drawer-container">
              <div class="remediation-header">
                <span class="remediation-label">&#128736; Recommended Configuration Remediation</span>
                <button class="btn-copy-remediation" onclick="copyRemediationCode('${drawerId}-code', this)">
                  Copy Configuration
                </button>
              </div>
              <div class="remediation-code-block" id="${drawerId}-code">${escapeHTML(f.remediation)}</div>
            </div>
          </td>
        </tr>
      `;
    }

    return rowHTML;
  }).join("");
}

window.toggleRemediationDrawer = function(drawerId) {
  const el = document.getElementById(drawerId);
  if (!el) return;
  if (el.style.display === "none" || el.style.display === "") {
    el.style.display = "table-row";
  } else {
    el.style.display = "none";
  }
};

window.copyRemediationCode = function(codeId, btnEl) {
  const codeEl = document.getElementById(codeId);
  if (!codeEl) return;
  const text = codeEl.innerText || codeEl.textContent;

  navigator.clipboard.writeText(text).then(() => {
    const originalText = btnEl.textContent;
    btnEl.textContent = "Copied!";
    btnEl.style.backgroundColor = "rgba(16, 185, 129, 0.2)";
    setTimeout(() => {
      btnEl.textContent = originalText;
      btnEl.style.backgroundColor = "";
    }, 2000);
  }).catch(err => {
    console.error("Clipboard copy failed:", err);
  });
};

function showAuditAlert(msg) {
  const el = document.getElementById("audit-alert-banner");
  if (!el) return;
  el.textContent = msg;
  el.style.display = "block";
}

function clearAuditAlert() {
  const el = document.getElementById("audit-alert-banner");
  if (!el) return;
  el.textContent = "";
  el.style.display = "none";
}

function resetSecurityAuditState() {
  activeSecurityReport = null;
  const emptyState = document.getElementById("audit-empty-state");
  const resultsContainer = document.getElementById("audit-results-container");
  const btnExport = document.getElementById("btn-export-security-report");

  if (emptyState) emptyState.style.display = "flex";
  if (resultsContainer) resultsContainer.style.display = "none";
  if (btnExport) btnExport.style.display = "none";
  clearAuditAlert();
}

// ==========================================================================
// Dual-Engine Hero Showcase Tabs (TASK-012)
// ==========================================================================
function initShowcaseTabs() {
  const tabTelemetry = document.getElementById("tab-telemetry-btn");
  const tabSecurity = document.getElementById("tab-security-btn");
  const panelTelemetry = document.getElementById("showcase-telemetry-panel");
  const panelSecurity = document.getElementById("showcase-security-panel");
  const headerTitle = document.getElementById("showcase-header-title");

  if (!tabTelemetry || !tabSecurity || !panelTelemetry || !panelSecurity) {
    return;
  }

  tabTelemetry.addEventListener("click", () => {
    tabTelemetry.classList.add("active");
    tabSecurity.classList.remove("active");
    panelTelemetry.style.display = "block";
    panelSecurity.style.display = "none";
    if (headerTitle) {
      headerTitle.textContent = "GoodAI Telemetry Console \u2022 Global Distributed Edge Network";
    }
  });

  tabSecurity.addEventListener("click", () => {
    tabSecurity.classList.add("active");
    tabTelemetry.classList.remove("active");
    panelTelemetry.style.display = "none";
    panelSecurity.style.display = "block";
    if (headerTitle) {
      headerTitle.textContent = "Web Security & AI Defense \u2022 Multi-Vector Attack Surface Audit";
    }
  });
}

// ==========================================================================
// Instant Public Inspection Playground (TASK-012)
// ==========================================================================
function initPublicAudit() {
  const form = document.getElementById("public-audit-form");
  const urlInput = document.getElementById("public-audit-url");
  const submitBtn = document.getElementById("btn-public-audit-submit");
  const loadingEl = document.getElementById("public-audit-loading");
  const errorEl = document.getElementById("public-audit-error");
  const resultEl = document.getElementById("public-audit-result");

  if (!form || !urlInput || !submitBtn || !resultEl) {
    return;
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();

    let target = urlInput.value.trim();
    if (!target) return;

    // Reset view states
    if (errorEl) {
      errorEl.textContent = "";
      errorEl.style.display = "none";
    }
    resultEl.style.display = "none";
    if (loadingEl) loadingEl.style.display = "flex";
    submitBtn.disabled = true;

    try {
      const res = await fetch("/api/v1/security/public-audit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_url: target })
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        throw new Error(data.error || `Inspection failed (HTTP ${res.status})`);
      }

      renderPublicAuditResult(data.report, resultEl);
      resultEl.style.display = "block";
    } catch (err) {
      if (errorEl) {
        errorEl.textContent = err.message || "Failed to complete security inspection.";
        errorEl.style.display = "block";
      }
    } finally {
      if (loadingEl) loadingEl.style.display = "none";
      submitBtn.disabled = false;
    }
  });
}

function renderPublicAuditResult(report, container) {
  if (!report) return;

  const score = report.composite_score || 0;
  const grade = report.grade || "B";
  const host = escapeHTML(report.target_host || "target");
  const execMs = report.execution_ms || 180;

  let gradeColor = "#10b981"; // green
  if (grade === "F") gradeColor = "#ef4444";
  else if (grade === "C") gradeColor = "#f59e0b";
  else if (grade === "B") gradeColor = "#3b82f6";

  const domains = report.domain_scores || [];
  let domainCardsHtml = "";
  domains.forEach(d => {
    domainCardsHtml += `
      <div class="audit-domain-box-sm">
        <span class="domain-name">${escapeHTML(d.domain_name)}</span>
        <span class="domain-score">${d.score}/${d.max_score}</span>
        <span style="font-size: 0.6875rem; color: var(--text-muted);">${escapeHTML(d.status || "")}</span>
      </div>
    `;
  });

  const findings = report.findings || [];
  let previewFindingsHtml = "";
  const keyFindings = findings.slice(0, 3);
  keyFindings.forEach(f => {
    let sevBadge = `<span class="sec-tag-chip verified">${escapeHTML(f.severity)}</span>`;
    if (f.severity === "CRITICAL" || f.severity === "HIGH") {
      sevBadge = `<span class="sec-tag-chip" style="background-color: rgba(239, 68, 68, 0.15); color: #ef4444; border-color: rgba(239, 68, 68, 0.3);">${escapeHTML(f.severity)}</span>`;
    } else if (f.severity === "MEDIUM") {
      sevBadge = `<span class="sec-tag-chip" style="background-color: rgba(245, 158, 11, 0.15); color: #f59e0b; border-color: rgba(245, 158, 11, 0.3);">${escapeHTML(f.severity)}</span>`;
    }
    previewFindingsHtml += `
      <div class="audit-finding-preview-row">
        <span><strong>${escapeHTML(f.check_name)}</strong>: ${escapeHTML(f.observed_value || "")}</span>
        ${sevBadge}
      </div>
    `;
  });

  container.innerHTML = `
    <div class="audit-hero-banner">
      <div class="audit-hero-info">
        <div class="audit-hero-title">
          <span>&#128737;</span>
          <span>Security Audit for ${host}</span>
        </div>
        <div class="audit-hero-meta">
          <span>Execution: ${execMs}ms &bull; Zero-Trust Prober &bull; Protocol Verified</span>
        </div>
      </div>
      <div class="audit-score-badge-large">
        <div style="display: flex; flex-direction: column; align-items: flex-end;">
          <span style="font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;">Composite Score</span>
          <span style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: ${gradeColor};">${score}/100</span>
        </div>
        <span class="audit-grade-pill" style="background-color: ${gradeColor}; color: #ffffff;">${escapeHTML(grade)}</span>
      </div>
    </div>

    <div class="audit-domains-grid-sm">
      ${domainCardsHtml}
    </div>

    ${previewFindingsHtml ? `
      <div class="audit-findings-preview-list">
        <div style="font-size: 0.75rem; font-weight: 600; color: var(--text-muted); margin-bottom: 0.25rem; text-transform: uppercase;">
          Sample Inspection Findings
        </div>
        ${previewFindingsHtml}
      </div>
    ` : ""}

    <div class="audit-cta-row">
      <div class="audit-cta-text">
        <span>Want continuous 10s probing, historical SLA trends, and automated PDF executive reports?</span>
      </div>
      <button type="button" class="btn-playground-cta" id="btn-playground-signup">
        Create Free Account &rarr;
      </button>
    </div>
  `;

  const btnSignup = document.getElementById("btn-playground-signup");
  if (btnSignup) {
    btnSignup.addEventListener("click", () => {
      const btnOpenRegister = document.getElementById("btn-open-register");
      if (btnOpenRegister) {
        btnOpenRegister.click();
      }
    });
  }
}

// ==========================================================================
// Developer FAQ Accordion Management (TASK-012)
// ==========================================================================
function initFAQ() {
  const faqItems = document.querySelectorAll(".faq-item");
  faqItems.forEach(item => {
    item.addEventListener("toggle", () => {
      if (item.open) {
        faqItems.forEach(other => {
          if (other !== item && other.open) {
            other.removeAttribute("open");
          }
        });
      }
    });
  });
}


