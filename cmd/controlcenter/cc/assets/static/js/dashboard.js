/* ============================================================
   Orkestra Control Center – dashboard.js
   Shared: theme toggle, sidebar, mobile nav, SSE live dot
   Loaded by: katalog.html, crd.html, cr_list.html, cr_detail.html, metrics.html
   ============================================================ */
(function () {
  'use strict';

  /* ── 1. Dark / Light theme toggle ─────────────────────────── */
  var THEME_KEY = 'cc-theme';

  function getTheme() {
    return localStorage.getItem(THEME_KEY) || 'dark';
  }

  function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(THEME_KEY, theme);
    var btn = document.getElementById('cc-theme-toggle');
    if (btn) {
      btn.textContent = theme === 'dark' ? '☀️' : '🌙';
      btn.title = theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode';
    }
  }

  // Apply immediately (the inline <head> script already did this, but update button label)
  applyTheme(getTheme());

  var themeToggle = document.getElementById('cc-theme-toggle');
  if (themeToggle) {
    themeToggle.addEventListener('click', function () {
      applyTheme(getTheme() === 'dark' ? 'light' : 'dark');
    });
  }

  /* ── 2. Sidebar: mobile toggle + collapsible ───────────────── */
  var mobileToggle = document.getElementById('cc-mobile-toggle');
  var sidebar = document.getElementById('cc-sidebar');
  var overlay = document.getElementById('cc-overlay');
  var collapseToggle = document.getElementById('cc-sidebar-toggle');

  // Restore collapsed state
  if (sidebar && localStorage.getItem('cc-sidebar-collapsed') === '1') {
    sidebar.classList.add('collapsed');
  }

  function openSidebar() {
    if (sidebar) sidebar.classList.add('open');
    if (overlay) overlay.classList.add('open');
  }

  function closeSidebar() {
    if (sidebar) sidebar.classList.remove('open');
    if (overlay) overlay.classList.remove('open');
  }

  if (mobileToggle) mobileToggle.addEventListener('click', openSidebar);
  if (overlay) overlay.addEventListener('click', closeSidebar);

  if (collapseToggle && sidebar) {
    collapseToggle.addEventListener('click', function () {
      var collapsed = sidebar.classList.toggle('collapsed');
      localStorage.setItem('cc-sidebar-collapsed', collapsed ? '1' : '0');
    });
  }

  /* ── 3. SSE live dot ───────────────────────────────────────── */
  var sseDot = document.getElementById('sse-dot');
  var sseRetryDelay = 1000;
  var sseMaxRetry = 30000;
  var sseSource = null;
  var sseDisconnectedBanner = null;
  var SSE_BANNER_THRESHOLD = 4;
  var sseFailCount = 0;

  function setSseConnected(connected) {
    if (connected) { sseFailCount = 0; }
    if (sseDot) {
      sseDot.classList.toggle('connected', connected);
    }
    if (!connected) {
      sseFailCount++;
      if (sseFailCount >= SSE_BANNER_THRESHOLD && !sseDisconnectedBanner) {
        sseDisconnectedBanner = document.createElement('div');
        sseDisconnectedBanner.className = 'cc-alert cc-alert-warn cc-disconnected-banner';
        sseDisconnectedBanner.innerHTML =
          '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>' +
          '<span>Disconnected from runtime — data may be stale</span>';
        var page = document.querySelector('.cc-page');
        if (page) page.insertBefore(sseDisconnectedBanner, page.firstChild);
      }
    } else {
      if (sseDisconnectedBanner) {
        sseDisconnectedBanner.remove();
        sseDisconnectedBanner = null;
      }
    }
  }

  function connectSSE() {
    if (sseSource) { sseSource.close(); sseSource = null; }
    // Use a page-appropriate SSE endpoint — fall back to generic /controlcenter/events
    var sseUrl = (window.CC_SSE_URL || '/controlcenter/events');
    try { sseSource = new EventSource(sseUrl); } catch (e) { scheduleReconnect(); return; }

    sseSource.onopen = function () {
      setSseConnected(true);
      sseRetryDelay = 1000;
    };

    sseSource.onmessage = function (e) {
      if (e.data === 'reload') { window.location.reload(); }
    };

    sseSource.onerror = function () {
      setSseConnected(false);
      sseSource.close();
      sseSource = null;
      scheduleReconnect();
    };
  }

  function scheduleReconnect() {
    setTimeout(connectSSE, sseRetryDelay);
    sseRetryDelay = Math.min(sseRetryDelay * 2, sseMaxRetry);
  }

  connectSSE();

})();
