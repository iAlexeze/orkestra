/* ============================================================
   manageRuntime.js — Runtime Management Drawer
   Single panel: list + inline add + inline edit + delete confirm
   No page reloads. Async throughout.
   ============================================================ */
(function () {
  'use strict';

  /* ── Elements ──────────────────────────────────────────────── */
  var drawer      = document.getElementById('runtimeDrawer');
  var backdrop    = document.getElementById('rtBackdrop');
  var closeBtn    = document.getElementById('rtCloseBtn');
  var openBtns    = [
    document.getElementById('manageRuntimesBtn'),
    document.getElementById('sidebarRuntimesBtn'),
  ];
  var addInput    = document.getElementById('rtAddInput');
  var addBtn      = document.getElementById('rtAddBtn');
  var addMsg      = document.getElementById('rtAddMsg');
  var listEl      = document.getElementById('rtList');
  var countEl     = document.getElementById('rtCount');

  if (!drawer) return; // not on index page

  /* ── Drawer open / close ───────────────────────────────────── */
  function openDrawer() {
    drawer.hidden = false;
    requestAnimationFrame(function () { drawer.classList.add('open'); });
    loadList();
    if (addInput) addInput.focus();
  }

  function closeDrawer() {
    drawer.classList.remove('open');
    drawer.addEventListener('transitionend', function handler() {
      drawer.hidden = true;
      drawer.removeEventListener('transitionend', handler);
    }, { once: true });
  }

  openBtns.forEach(function (btn) {
    if (btn) btn.addEventListener('click', function (e) { e.preventDefault(); openDrawer(); });
  });
  if (closeBtn)  closeBtn.addEventListener('click', closeDrawer);
  if (backdrop)  backdrop.addEventListener('click', closeDrawer);
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && !drawer.hidden) closeDrawer();
  });

  /* ── Helpers ───────────────────────────────────────────────── */
  function esc(str) {
    return String(str || '')
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  function showMsg(el, text, type) {
    if (!el) return;
    el.textContent = text;
    el.className = 'rt-msg rt-msg-' + type;
    el.hidden = false;
    if (type === 'success') setTimeout(function () { el.hidden = true; }, 3000);
  }

  function hideMsg(el) { if (el) el.hidden = true; }

  function setLoading(btn, loading) {
    if (!btn) return;
    btn.disabled = loading;
    btn.classList.toggle('loading', loading);
  }

  /* ── Load runtime list ─────────────────────────────────────── */
  function loadList() {
    if (!listEl) return;
    listEl.innerHTML = '<div class="rt-list-loading"><svg class="rt-spinner" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>Loading…</div>';

    fetch('/controlcenter/api/instances', { cache: 'no-store' })
      .then(function (r) {
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.json();
      })
      .then(function (data) {
        renderList(data.urls || []);
      })
      .catch(function (err) {
        listEl.innerHTML = '<div class="rt-list-empty rt-list-error"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>Failed to load: ' + esc(err.message) + '</div>';
      });
  }

  /* ── Render list ───────────────────────────────────────────── */
  function renderList(urls) {
    if (countEl) countEl.textContent = urls.length + (urls.length === 1 ? ' runtime' : ' runtimes');

    if (!urls.length) {
      listEl.innerHTML = '<div class="rt-list-empty"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16" stroke-dasharray="2 2"/></svg><span>No runtimes configured.<br>Add one above.</span></div>';
      return;
    }

    var html = '';
    urls.forEach(function (url, idx) {
      html += '<div class="rt-item" data-url="' + esc(url) + '" id="rt-item-' + idx + '">';
      html += '  <div class="rt-item-left">';
      html += '    <span class="rt-item-dot" id="rt-dot-' + idx + '"></span>';
      html += '    <div class="rt-item-info">';
      html += '      <span class="rt-item-url" id="rt-url-' + idx + '">' + esc(url) + '</span>';
      html += '      <span class="rt-item-status" id="rt-status-' + idx + '">checking…</span>';
      html += '    </div>';
      html += '  </div>';
      html += '  <div class="rt-item-actions">';
      html += '    <button class="rt-action-btn rt-edit-btn" data-url="' + esc(url) + '" title="Edit URL" aria-label="Edit">';
      html += '      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>';
      html += '    </button>';
      html += '    <button class="rt-action-btn rt-delete-btn" data-url="' + esc(url) + '" title="Remove" aria-label="Remove">';
      html += '      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>';
      html += '    </button>';
      html += '  </div>';
      html += '</div>';
    });

    listEl.innerHTML = html;

    // Attach edit / delete handlers
    listEl.querySelectorAll('.rt-edit-btn').forEach(function (btn) {
      btn.addEventListener('click', function () { startInlineEdit(btn.dataset.url); });
    });
    listEl.querySelectorAll('.rt-delete-btn').forEach(function (btn) {
      btn.addEventListener('click', function () { confirmDelete(btn.dataset.url, btn); });
    });

    // Async health check per runtime
    urls.forEach(function (url, idx) {
      checkHealth(url, idx);
    });
  }

  /* ── Health check ──────────────────────────────────────────── */
  function checkHealth(url, idx) {
    var dot    = document.getElementById('rt-dot-' + idx);
    var status = document.getElementById('rt-status-' + idx);
    var healthUrl = url.replace(/\/$/, '') + '/health';

    fetch(healthUrl, { cache: 'no-store', signal: AbortSignal.timeout(5000) })
      .then(function (r) {
        if (!dot || !status) return;
        if (r.ok) {
          dot.className = 'rt-item-dot online';
          status.textContent = 'online';
          status.className = 'rt-item-status rt-status-online';
        } else {
          dot.className = 'rt-item-dot offline';
          status.textContent = 'HTTP ' + r.status;
          status.className = 'rt-item-status rt-status-offline';
        }
      })
      .catch(function () {
        if (!dot || !status) return;
        dot.className = 'rt-item-dot offline';
        status.textContent = 'unreachable';
        status.className = 'rt-item-status rt-status-offline';
      });
  }

  /* ── Add runtime ───────────────────────────────────────────── */
  function normalizeUrl(val) {
    val = val.trim();
    if (val && !val.startsWith('http://') && !val.startsWith('https://')) {
      val = 'http://' + val;
    }
    return val;
  }

  function addRuntime() {
    var url = normalizeUrl(addInput ? addInput.value : '');
    if (!url) { showMsg(addMsg, 'Enter a URL first.', 'error'); return; }

    hideMsg(addMsg);
    setLoading(addBtn, true);

    fetch('/controlcenter/api/instances', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: url }),
    })
      .then(function (r) {
        return r.json().then(function (d) { return { ok: r.ok, data: d }; });
      })
      .then(function (res) {
        if (!res.ok) throw new Error(res.data.error || 'Add failed');
        if (addInput) addInput.value = '';
        showMsg(addMsg, 'Runtime added.', 'success');
        loadList();
      })
      .catch(function (err) {
        showMsg(addMsg, err.message, 'error');
      })
      .finally(function () { setLoading(addBtn, false); });
  }

  if (addBtn) addBtn.addEventListener('click', addRuntime);
  if (addInput) {
    addInput.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') addRuntime();
    });
  }

  /* ── Inline edit ───────────────────────────────────────────── */
  function startInlineEdit(originalUrl) {
    var item = listEl.querySelector('[data-url="' + CSS.escape(originalUrl) + '"]');
    if (!item) return;

    // Replace row content with inline edit form
    var saved = item.innerHTML;
    item.classList.add('editing');
    item.innerHTML =
      '<div class="rt-inline-edit">' +
      '  <input type="text" class="rt-input rt-inline-input" value="' + esc(originalUrl) + '" autocomplete="off">' +
      '  <button class="cc-btn cc-btn-primary rt-inline-save">Save</button>' +
      '  <button class="cc-btn cc-btn-secondary rt-inline-cancel">Cancel</button>' +
      '</div>' +
      '<div class="rt-inline-msg" hidden></div>';

    var input  = item.querySelector('.rt-inline-input');
    var saveB  = item.querySelector('.rt-inline-save');
    var cancelB= item.querySelector('.rt-inline-cancel');
    var msgEl  = item.querySelector('.rt-inline-msg');

    if (input) { input.focus(); input.select(); }

    cancelB.addEventListener('click', function () {
      item.classList.remove('editing');
      item.innerHTML = saved;
      // Re-attach listeners on the restored row
      item.querySelector('.rt-edit-btn')  ?.addEventListener('click', function (e) { startInlineEdit(e.currentTarget.dataset.url); });
      item.querySelector('.rt-delete-btn')?.addEventListener('click', function (e) { confirmDelete(e.currentTarget.dataset.url, e.currentTarget); });
    });

    function doSave() {
      var newUrl = normalizeUrl(input.value);
      if (!newUrl) { showMsg(msgEl, 'URL cannot be empty.', 'error'); return; }
      setLoading(saveB, true);

      fetch('/controlcenter/api/instances/' + encodeURIComponent(originalUrl), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: newUrl }),
      })
        .then(function (r) {
          return r.json().then(function (d) { return { ok: r.ok, data: d }; });
        })
        .then(function (res) {
          if (!res.ok) throw new Error(res.data.error || 'Update failed');
          loadList();
        })
        .catch(function (err) {
          showMsg(msgEl, err.message, 'error');
          setLoading(saveB, false);
        });
    }

    saveB.addEventListener('click', doSave);
    input.addEventListener('keydown', function (e) { if (e.key === 'Enter') doSave(); });
  }

  /* ── Delete with inline confirm ────────────────────────────── */
  function confirmDelete(url, btn) {
    var item = btn.closest('.rt-item');
    if (!item) return;

    // Swap delete button for confirm/cancel pair inline
    var actionsEl = item.querySelector('.rt-item-actions');
    if (!actionsEl) return;
    var original = actionsEl.innerHTML;

    actionsEl.innerHTML =
      '<span class="rt-confirm-text">Remove?</span>' +
      '<button class="rt-action-btn rt-confirm-yes">Yes</button>' +
      '<button class="rt-action-btn rt-confirm-no">No</button>';

    actionsEl.querySelector('.rt-confirm-no').addEventListener('click', function () {
      actionsEl.innerHTML = original;
      actionsEl.querySelector('.rt-edit-btn')  ?.addEventListener('click', function (e) { startInlineEdit(e.currentTarget.dataset.url); });
      actionsEl.querySelector('.rt-delete-btn')?.addEventListener('click', function (e) { confirmDelete(e.currentTarget.dataset.url, e.currentTarget); });
    });

    actionsEl.querySelector('.rt-confirm-yes').addEventListener('click', function () {
      var yesBtn = actionsEl.querySelector('.rt-confirm-yes');
      setLoading(yesBtn, true);
      item.classList.add('deleting');

      fetch('/controlcenter/api/instances/' + encodeURIComponent(url), { method: 'DELETE' })
        .then(function (r) {
          return r.json().then(function (d) { return { ok: r.ok, data: d }; });
        })
        .then(function (res) {
          if (!res.ok) throw new Error(res.data.error || 'Delete failed');
          // Animate out, then reload list
          item.style.transition = 'opacity 0.2s, transform 0.2s';
          item.style.opacity = '0';
          item.style.transform = 'translateX(24px)';
          setTimeout(loadList, 220);
        })
        .catch(function (err) {
          item.classList.remove('deleting');
          actionsEl.innerHTML = original;
          actionsEl.querySelector('.rt-edit-btn')  ?.addEventListener('click', function (e) { startInlineEdit(e.currentTarget.dataset.url); });
          actionsEl.querySelector('.rt-delete-btn')?.addEventListener('click', function (e) { confirmDelete(e.currentTarget.dataset.url, e.currentTarget); });
          // Show brief error on item
          var errSpan = document.createElement('span');
          errSpan.className = 'rt-item-err';
          errSpan.textContent = err.message;
          item.appendChild(errSpan);
          setTimeout(function () { errSpan.remove(); }, 3000);
        });
    });
  }

})();
