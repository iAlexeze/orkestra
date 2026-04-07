// manageRuntime.js
// Handles the "Manage Runtimes" modal: list, add, edit, delete runtime URLs.
// Backend API endpoints:
//   GET    /controlcenter/api/instances
//   POST   /controlcenter/api/instances
//   PUT    /controlcenter/api/instances/:url
//   DELETE /controlcenter/api/instances/:url
// 
(function() {
    // DOM elements
    const manageBtn = document.getElementById('manageRuntimesBtn');
    const manageModal = document.getElementById('manageRuntimesModal');
    const closeManageModalBtn = document.getElementById('closeManageModalBtn');
    const closeManageFooterBtn = document.getElementById('closeManageFooterBtn');
    const runtimeListContainer = document.getElementById('runtimeListContainer');
    const manageErrorDiv = document.getElementById('manageError');
    const openAddFromManageBtn = document.getElementById('openAddFromManageBtn');

    // Add/Edit modal elements
    const runtimeModal = document.getElementById('runtimeModal');
    const runtimeModalTitle = document.getElementById('runtimeModalTitle');
    const runtimeUrlInput = document.getElementById('runtimeUrlInput');
    const runtimeModalError = document.getElementById('runtimeModalError');
    const closeRuntimeModalBtn = document.getElementById('closeRuntimeModalBtn');
    const cancelRuntimeBtn = document.getElementById('cancelRuntimeBtn');
    const saveRuntimeBtn = document.getElementById('saveRuntimeBtn');

    // Delete confirmation modal
    const deleteModal = document.getElementById('deleteConfirmModal');
    const deleteRuntimeUrlSpan = document.getElementById('deleteRuntimeUrl');
    const closeDeleteModalBtn = document.getElementById('closeDeleteModalBtn');
    const cancelDeleteBtn = document.getElementById('cancelDeleteBtn');
    const confirmDeleteBtn = document.getElementById('confirmDeleteBtn');

    // State
    let currentEditUrl = null;   // if editing, store original URL
    let pendingDeleteUrl = null;

    // Helper: escape HTML
    function escapeHtml(str) {
        if (!str) return '';
        return str.replace(/[&<>]/g, function(m) {
            if (m === '&') return '&amp;';
            if (m === '<') return '&lt;';
            if (m === '>') return '&gt;';
            return m;
        });
    }

    // Show/hide modals
    function showModal(modal) { if (modal) modal.style.display = 'flex'; }
    function hideModal(modal) { if (modal) modal.style.display = 'none'; }

    // Show error in a modal
    function showError(container, message) {
        if (!container) return;
        container.textContent = message;
        container.style.display = 'block';
        setTimeout(() => container.style.display = 'none', 4000);
    }

    // Fetch and render runtime list inside manage modal
    async function loadRuntimeList() {
        if (!runtimeListContainer) return;
        runtimeListContainer.innerHTML = '<div class="text-center text-muted">Loading...</div>';
        try {
            const resp = await fetch('/controlcenter/api/instances');
            if (!resp.ok) throw new Error('Failed to fetch runtimes');
            const data = await resp.json();
            const urls = data.urls || [];
            if (urls.length === 0) {
                runtimeListContainer.innerHTML = '<div class="text-center text-muted">No runtimes configured.</div>';
                return;
            }
            let html = '<div class="runtime-list">';
            urls.forEach(url => {
                // You could optionally check health status here (extra fetch)
                html += `
                    <div class="runtime-item" data-url="${escapeHtml(url)}">
                        <div class="runtime-url">${escapeHtml(url)}</div>
                        <div class="runtime-actions">
                            <button class="edit-runtime-btn cc-btn cc-btn-sm" data-url="${escapeHtml(url)}">Edit</button>
                            <button class="delete-runtime-btn cc-btn cc-btn-sm cc-btn-danger" data-url="${escapeHtml(url)}">Delete</button>
                        </div>
                    </div>
                `;
            });
            html += '</div>';
            runtimeListContainer.innerHTML = html;
            // Attach event listeners
            document.querySelectorAll('.edit-runtime-btn').forEach(btn => {
                btn.addEventListener('click', () => openEditModal(btn.dataset.url));
            });
            document.querySelectorAll('.delete-runtime-btn').forEach(btn => {
                btn.addEventListener('click', () => openDeleteModal(btn.dataset.url));
            });
        } catch (err) {
            runtimeListContainer.innerHTML = `<div class="text-center text-error">Error: ${escapeHtml(err.message)}</div>`;
        }
    }

    // Open Add/Edit modal
    function openAddModal() {
        currentEditUrl = null;
        runtimeModalTitle.textContent = 'Add Orkestra Runtime';
        runtimeUrlInput.value = '';
        hideModal(runtimeModalError);
        showModal(runtimeModal);
    }

    function openEditModal(url) {
        currentEditUrl = url;
        runtimeModalTitle.textContent = 'Edit Runtime';
        runtimeUrlInput.value = url;
        hideModal(runtimeModalError);
        showModal(runtimeModal);
    }

    // Save runtime (add or edit)
    async function saveRuntime() {
        let url = runtimeUrlInput.value.trim();
        if (!url) {
            showError(runtimeModalError, 'URL is required');
            return;
        }
        if (!url.startsWith('http://') && !url.startsWith('https://')) {
            url = 'http://' + url;
        }
        const method = currentEditUrl ? 'PUT' : 'POST';
        const endpoint = currentEditUrl
            ? `/controlcenter/api/instances/${encodeURIComponent(currentEditUrl)}`
            : '/controlcenter/api/instances';
        const body = JSON.stringify({ url: url });

        saveRuntimeBtn.disabled = true;
        saveRuntimeBtn.textContent = 'Saving...';
        try {
            const resp = await fetch(endpoint, { method, headers: { 'Content-Type': 'application/json' }, body });
            if (!resp.ok) {
                const err = await resp.json();
                throw new Error(err.error || 'Operation failed');
            }
            // Success: reload page to reflect changes
            window.location.reload();
        } catch (err) {
            showError(runtimeModalError, err.message);
            saveRuntimeBtn.disabled = false;
            saveRuntimeBtn.textContent = 'Save';
        }
    }

    // Open delete confirmation modal
    function openDeleteModal(url) {
        pendingDeleteUrl = url;
        deleteRuntimeUrlSpan.textContent = url;
        showModal(deleteModal);
    }

    async function confirmDelete() {
        if (!pendingDeleteUrl) return;
        confirmDeleteBtn.disabled = true;
        confirmDeleteBtn.textContent = 'Deleting...';
        try {
            const resp = await fetch(`/controlcenter/api/instances/${encodeURIComponent(pendingDeleteUrl)}`, { method: 'DELETE' });
            if (!resp.ok) {
                const err = await resp.json();
                throw new Error(err.error || 'Delete failed');
            }
            window.location.reload();
        } catch (err) {
            showError(manageErrorDiv, err.message);
            confirmDeleteBtn.disabled = false;
            confirmDeleteBtn.textContent = 'Delete';
            hideModal(deleteModal);
        }
    }

    // Modal controls
    function showManageModal() {
        loadRuntimeList();
        showModal(manageModal);
    }
    function hideManageModal() { hideModal(manageModal); }

    // Event listeners
    if (manageBtn) manageBtn.addEventListener('click', showManageModal);
    if (closeManageModalBtn) closeManageModalBtn.addEventListener('click', hideManageModal);
    if (closeManageFooterBtn) closeManageFooterBtn.addEventListener('click', hideManageModal);
    if (manageModal) manageModal.addEventListener('click', (e) => { if (e.target === manageModal) hideManageModal(); });

    // Add/Edit modal listeners
    if (openAddFromManageBtn) openAddFromManageBtn.addEventListener('click', () => {
        hideManageModal();
        openAddModal();
    });
    if (closeRuntimeModalBtn) closeRuntimeModalBtn.addEventListener('click', () => hideModal(runtimeModal));
    if (cancelRuntimeBtn) cancelRuntimeBtn.addEventListener('click', () => hideModal(runtimeModal));
    if (saveRuntimeBtn) saveRuntimeBtn.addEventListener('click', saveRuntime);
    if (runtimeModal) runtimeModal.addEventListener('click', (e) => { if (e.target === runtimeModal) hideModal(runtimeModal); });

    // Delete modal listeners
    if (closeDeleteModalBtn) closeDeleteModalBtn.addEventListener('click', () => hideModal(deleteModal));
    if (cancelDeleteBtn) cancelDeleteBtn.addEventListener('click', () => hideModal(deleteModal));
    if (confirmDeleteBtn) confirmDeleteBtn.addEventListener('click', confirmDelete);
    if (deleteModal) deleteModal.addEventListener('click', (e) => { if (e.target === deleteModal) hideModal(deleteModal); });

    // Top‑right "Add Runtime" button (re‑enable)
    const addRuntimeTopBtn = document.getElementById('addRuntimeBtn');
    if (addRuntimeTopBtn) {
        addRuntimeTopBtn.addEventListener('click', openAddModal);
    }
})();