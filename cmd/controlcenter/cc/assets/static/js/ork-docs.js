// Orkestra CC — docs page JS
(function () {
  // Theme toggle
  var themeBtn = document.getElementById('ork-theme-toggle');
  if (themeBtn) {
    themeBtn.addEventListener('click', function () {
      var t = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
      document.documentElement.setAttribute('data-theme', t);
      localStorage.setItem('cc-theme', t);
    });
  }

  // Sidebar group toggles
  document.querySelectorAll('.ork-sidebar-group-toggle').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var group = btn.closest('.ork-sidebar-group');
      if (group) group.classList.toggle('is-open');
    });
  });

  // Highlight active sidebar link based on scroll position
  var headings = document.querySelectorAll('.ork-doc-body h2[id], .ork-doc-body h3[id]');
  var sidebarLinks = document.querySelectorAll('.ork-sidebar-item a[href^="#"]');
  if (headings.length && sidebarLinks.length) {
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          sidebarLinks.forEach(function (a) { a.closest('.ork-sidebar-item').classList.remove('active'); });
          var match = document.querySelector('.ork-sidebar-item a[href="#' + entry.target.id + '"]');
          if (match) match.closest('.ork-sidebar-item').classList.add('active');
        }
      });
    }, { rootMargin: '-20% 0px -70% 0px' });
    headings.forEach(function (h) { observer.observe(h); });
  }
})();
