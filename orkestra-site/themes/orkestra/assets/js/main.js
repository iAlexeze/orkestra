/* ── Theme toggle ─────────────────────────────────────────────────────────── */
(function () {
  const STORAGE_KEY = 'orkestra-theme';
  const root = document.documentElement;

  function applyTheme(theme) {
    root.setAttribute('data-theme', theme);
    localStorage.setItem(STORAGE_KEY, theme);
    document.querySelectorAll('.theme-toggle').forEach(btn => {
      btn.setAttribute('aria-label', `Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`);
      btn.setAttribute('aria-pressed', theme === 'dark' ? 'true' : 'false');
    });
  }

  function getPreferred() {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) return stored;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  applyTheme(getPreferred());

  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
    if (!localStorage.getItem(STORAGE_KEY)) applyTheme(e.matches ? 'dark' : 'light');
  });

  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.theme-toggle').forEach(btn => {
      btn.addEventListener('click', () => {
        const current = root.getAttribute('data-theme') || 'dark';
        applyTheme(current === 'dark' ? 'light' : 'dark');
      });
    });
  });
})();

/* ── Mobile nav ───────────────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  const toggle = document.getElementById('hamburger');
  const navLinks = document.getElementById('mobileMenu');

  if (toggle && navLinks) {
    toggle.addEventListener('click', () => {
      const open = navLinks.classList.toggle('open');
      toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
    });

    // Close on outside click
    document.addEventListener('click', e => {
      if (!toggle.contains(e.target) && !navLinks.contains(e.target)) {
        navLinks.classList.remove('open');
        toggle.setAttribute('aria-expanded', 'false');
      }
    });
  }
});

/* ── Sidebar mobile toggle ────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  const sidebarToggle = document.getElementById('sidebar-toggle');
  const sidebar = document.getElementById('sidebar');

  if (sidebarToggle && sidebar) {
    sidebarToggle.addEventListener('click', () => {
      const open = sidebar.classList.toggle('open');
      sidebarToggle.setAttribute('aria-expanded', open ? 'true' : 'false');
    });

    document.addEventListener('click', e => {
      if (!sidebarToggle.contains(e.target) && !sidebar.contains(e.target)) {
        sidebar.classList.remove('open');
        sidebarToggle.setAttribute('aria-expanded', 'false');
      }
    });
  }
});

/* ── Sidebar tree toggles ─────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.sidebar-section-title').forEach(title => {
    const ul = title.nextElementSibling;
    if (!ul || ul.tagName !== 'UL') return;

    title.setAttribute('role', 'button');
    title.setAttribute('tabindex', '0');

    const toggle = () => {
      const collapsed = ul.classList.toggle('collapsed');
      title.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    };

    title.addEventListener('click', toggle);
    title.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggle(); }
    });

    // Auto-expand section containing current page
    if (ul.querySelector('.active')) {
      ul.classList.remove('collapsed');
      title.setAttribute('aria-expanded', 'true');
    } else {
      ul.classList.add('collapsed');
      title.setAttribute('aria-expanded', 'false');
    }
  });
});

/* ── Active TOC link on scroll ────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  const tocLinks = document.querySelectorAll('.toc-list a');
  if (!tocLinks.length) return;

  const headings = Array.from(tocLinks).map(a => {
    const id = a.getAttribute('href').replace('#', '');
    return { el: document.getElementById(id), link: a };
  }).filter(h => h.el);

  function updateActive() {
    const scrollY = window.scrollY + 96; // offset for sticky nav
    let active = headings[0];
    for (const h of headings) {
      if (h.el.offsetTop <= scrollY) active = h;
    }
    tocLinks.forEach(l => l.classList.remove('active'));
    if (active) active.link.classList.add('active');
  }

  window.addEventListener('scroll', updateActive, { passive: true });
  updateActive();
});

/* ── Mermaid ──────────────────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  if (typeof mermaid === 'undefined') return;
  const theme = document.documentElement.getAttribute('data-theme') === 'light' ? 'default' : 'dark';
  mermaid.initialize({ startOnLoad: false, theme, securityLevel: 'loose' });
  document.querySelectorAll('pre code.language-mermaid').forEach(block => {
    const pre = block.parentElement;
    const div = document.createElement('div');
    div.className = 'mermaid';
    div.textContent = block.textContent;
    pre.replaceWith(div);
  });
  mermaid.run();
});

/* ── Search (FlexSearch) ──────────────────────────────────────────────────── */
(function () {
  let index = null;
  let documents = [];
  let loaded = false;

  async function loadIndex() {
    if (loaded) return;
    loaded = true;
    try {
      const res = await fetch('/index.json');
      documents = await res.json();
      index = new FlexSearch.Document({
        document: { id: 'uri', index: ['title', 'content', 'tags'], store: ['title', 'uri', 'description'] },
        tokenize: 'forward',
        cache: 100,
      });
      documents.forEach(doc => index.add(doc));
    } catch (e) {
      console.warn('Search index failed to load', e);
    }
  }

  function highlight(text, query) {
    if (!text) return '';
    const re = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
    return text.replace(re, '<mark>$1</mark>');
  }

  function renderResults(query) {
    if (!index || !query.trim()) return '';
    const raw = index.search(query, { enrich: true, limit: 10 });
    const seen = new Set();
    const hits = raw.flatMap(r => r.result).filter(r => {
      if (seen.has(r.id)) return false;
      seen.add(r.id);
      return true;
    });
    if (!hits.length) return '<p class="search-empty">No results found.</p>';
    return hits.map(h => `
      <a href="${h.doc.uri}" class="search-result-item">
        <span class="search-result-title">${highlight(h.doc.title, query)}</span>
        ${h.doc.description ? `<span class="search-result-desc">${highlight(h.doc.description, query)}</span>` : ''}
      </a>`).join('');
  }

  document.addEventListener('DOMContentLoaded', () => {
    const modal = document.getElementById('searchModal');
    const input = document.getElementById('searchInput');
    const results = document.getElementById('searchResults');
    const openBtns = document.querySelectorAll('#searchTrigger, [data-search-open]');
    const closeBtn = document.getElementById('searchClose');

    if (!modal || !input || !results) return;

    function openModal() {
      modal.classList.add('open');
      input.focus();
      loadIndex();
    }

    function closeModal() {
      modal.classList.remove('open');
      input.value = '';
      results.innerHTML = '';
    }

    openBtns.forEach(btn => btn.addEventListener('click', openModal));
    closeBtn?.addEventListener('click', closeModal);
    modal.addEventListener('click', e => { if (e.target === modal) closeModal(); });

    document.addEventListener('keydown', e => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') { e.preventDefault(); openModal(); }
      if (e.key === 'Escape' && modal.classList.contains('open')) closeModal();
    });

    let debounceTimer;
    input.addEventListener('input', () => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        results.innerHTML = renderResults(input.value);
      }, 200);
    });

    // Keyboard nav in results
    input.addEventListener('keydown', e => {
      const items = results.querySelectorAll('.search-result-item');
      if (!items.length) return;
      const active = results.querySelector('.search-result-item:focus');
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        (active ? active.nextElementSibling || items[0] : items[0]).focus();
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        (active ? active.previousElementSibling || items[items.length - 1] : items[items.length - 1]).focus();
      }
    });
  });
})();

/* ── Copy code blocks ─────────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('pre').forEach(pre => {
    const btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.setAttribute('aria-label', 'Copy code');
    btn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
    </svg>`;

    btn.addEventListener('click', async () => {
      const code = pre.querySelector('code')?.textContent || pre.textContent;
      try {
        await navigator.clipboard.writeText(code);
        btn.classList.add('copied');
        btn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>`;
        setTimeout(() => {
          btn.classList.remove('copied');
          btn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>`;
        }, 2000);
      } catch (_) { /* clipboard not available */ }
    });

    pre.style.position = 'relative';
    pre.appendChild(btn);
  });
});

/* ── Anchor links for headings ────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.doc-body h2, .doc-body h3, .doc-body h4').forEach(h => {
    if (!h.id) return;
    const a = document.createElement('a');
    a.className = 'heading-anchor';
    a.href = `#${h.id}`;
    a.setAttribute('aria-label', 'Link to this section');
    a.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>`;
    h.appendChild(a);
  });
});

/* ── Smooth scroll for anchor links ──────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('a[href^="#"]').forEach(a => {
    a.addEventListener('click', e => {
      const id = a.getAttribute('href').slice(1);
      const target = document.getElementById(id);
      if (!target) return;
      e.preventDefault();
      target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      history.replaceState(null, '', `#${id}`);
    });
  });
});
