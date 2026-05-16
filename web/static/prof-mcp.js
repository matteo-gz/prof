(function () {
  'use strict';

  var BIN_KEY = 'prof:mcp:binary';

  function t(key) {
    return typeof window.t === 'function' ? window.t(key) : key;
  }

  function listenPort() {
    var p = window.location.port;
    if (p) return p;
    return window.location.protocol === 'https:' ? '443' : '80';
  }

  function apiBase() {
    return window.location.origin;
  }

  function isProfPath(s) {
    if (!s) return false;
    s = s.trim();
    if (s.charAt(0) === '{' || s.charAt(0) === '[') return false;
    if (s.indexOf('mcpServers') >= 0 || s.indexOf('"command"') >= 0) return false;
    return true;
  }

  function serverProfBin() {
    var body = document.body && document.body.dataset.profBin;
    if (isProfPath(body)) return body.trim();
    return '';
  }

  function profBin() {
    var el = document.getElementById('mcpProfBin');
    var v = el && el.value ? el.value.trim() : '';
    if (isProfPath(v)) return v;
    var detected = serverProfBin();
    if (detected) return detected;
    try {
      var saved = localStorage.getItem(BIN_KEY);
      if (isProfPath(saved)) return saved;
    } catch (e) {}
    return '/ABS/PATH/TO/prof';
  }

  function applyMcpHint(data) {
    if (!data) return;
    var bin = document.getElementById('mcpProfBin');
    if (bin && data.command && isProfPath(data.command)) {
      if (!bin.value.trim() || !isProfPath(bin.value.trim())) {
        bin.value = data.command;
      }
    }
  }

  function fetchMcpHint(done) {
    fetch(apiBase() + '/api/mcp-hint', { headers: { Accept: 'application/json' } })
      .then(function (r) {
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.json();
      })
      .then(function (data) {
        applyMcpHint(data);
        if (done) done();
      })
      .catch(function () {
        if (done) done();
      });
  }

  function setPathInvalid(invalid) {
    var bin = document.getElementById('mcpProfBin');
    var hint = document.getElementById('mcpPathHint');
    if (bin) bin.classList.toggle('is-invalid', invalid);
    if (hint) hint.style.display = invalid ? 'block' : 'none';
  }

  function buildMcpJSON() {
    return JSON.stringify(
      {
        mcpServers: {
          prof: {
            command: profBin(),
            args: ['-port', listenPort(), 'mcp'],
          },
        },
      },
      null,
      2
    );
  }

  function buildArgsJSON() {
    return JSON.stringify(['-port', listenPort(), 'mcp'], null, 2);
  }

  function applyI18n(root) {
    if (!root) return;
    root.querySelectorAll('[data-i18n]').forEach(function (el) {
      el.textContent = t(el.getAttribute('data-i18n'));
    });
    root.querySelectorAll('[data-i18n-placeholder]').forEach(function (el) {
      el.placeholder = t(el.getAttribute('data-i18n-placeholder'));
    });
  }

  function injectNav() {
    var nav = document.querySelector('.navbar-nav');
    if (!nav || document.getElementById('navUseMcp')) return;
    var li = document.createElement('li');
    li.className = 'nav-item';
    var a = document.createElement('a');
    a.className = 'nav-link';
    a.href = '#';
    a.id = 'navUseMcp';
    a.setAttribute('data-i18n', 'nav.useMcp');
    a.textContent = t('nav.useMcp');
    li.appendChild(a);
    nav.appendChild(li);
  }

  function injectModal() {
    if (document.getElementById('mcpModal')) return;
    var h = [];
    h.push('<div id="mcpModal" class="modal fade" tabindex="-1" style="display:none">');
    h.push('<div class="modal-dialog modal-lg"><div class="modal-content">');
    h.push('<div class="modal-header py-2">');
    h.push('<h6 class="modal-title mb-0" data-i18n="mcp.title"></h6>');
    h.push('<button type="button" class="btn-close" id="mcpModalClose" aria-label="Close"></button>');
    h.push('</div><div class="modal-body">');
    h.push('<p class="small text-muted mb-3" data-i18n="mcp.intro"></p>');
    h.push('<div class="mb-3 d-flex align-items-center gap-2 flex-wrap">');
    h.push('<span class="small text-muted" data-i18n="mcp.apiStatus"></span>');
    h.push('<span id="mcpApiBadge" class="badge bg-secondary">…</span>');
    h.push('<span class="small font-monospace text-muted" id="mcpApiBase"></span></div>');
    h.push('<label class="form-label small mb-1" data-i18n="mcp.profPath"></label>');
    h.push('<input type="text" class="form-control form-control-sm font-monospace mb-3" id="mcpProfBin" data-i18n-placeholder="mcp.profPathPlaceholder">');
    h.push('<label class="form-label small mb-1" data-i18n="mcp.configLabel"></label>');
    h.push('<pre id="mcpConfigContent" class="bg-light p-3 rounded small mb-3" style="max-height:280px;overflow:auto;white-space:pre"></pre>');
    h.push('<ol class="small text-muted mb-0 ps-3"><li data-i18n="mcp.step1"></li><li data-i18n="mcp.step2"></li><li data-i18n="mcp.step3"></li></ol>');
    h.push('</div><div class="modal-footer py-2">');
    h.push('<button type="button" class="btn btn-primary btn-sm" id="mcpCopyConfigBtn" data-i18n="mcp.copyConfig"></button>');
    h.push('<button type="button" class="btn btn-outline-secondary btn-sm" id="mcpCopyArgsBtn" data-i18n="mcp.copyArgs"></button>');
    h.push('<button type="button" class="btn btn-secondary btn-sm" id="mcpModalDismiss" data-i18n="btn.close"></button>');
    h.push('</div></div></div></div>');
    document.body.insertAdjacentHTML('beforeend', h.join(''));
    applyI18n(document.getElementById('mcpModal'));
  }

  function refreshMcpConfig() {
    var bin = document.getElementById('mcpProfBin');
    if (bin) {
      var v = bin.value.trim();
      if (v && !isProfPath(v)) {
        setPathInvalid(true);
        try {
          localStorage.removeItem(BIN_KEY);
        } catch (e) {}
      } else {
        setPathInvalid(false);
        if (v) {
          try {
            localStorage.setItem(BIN_KEY, v);
          } catch (e) {}
        }
      }
    }
    var pre = document.getElementById('mcpConfigContent');
    if (pre) pre.textContent = buildMcpJSON();
    var baseEl = document.getElementById('mcpApiBase');
    if (baseEl) baseEl.textContent = apiBase();
  }

  function checkAPI() {
    var badge = document.getElementById('mcpApiBadge');
    if (!badge) return;
    badge.className = 'badge bg-secondary';
    badge.textContent = '…';
    fetch(apiBase() + '/api/', { headers: { Accept: 'application/json' } })
      .then(function (r) {
        if (r.ok) {
          badge.className = 'badge bg-success';
          badge.textContent = t('mcp.apiOk');
        } else {
          badge.className = 'badge bg-warning text-dark';
          badge.textContent = 'HTTP ' + r.status;
        }
      })
      .catch(function () {
        badge.className = 'badge bg-danger';
        badge.textContent = t('mcp.apiDown');
      });
  }

  function showMcpModal() {
    var modal = document.getElementById('mcpModal');
    if (!modal) return;
    modal.style.display = 'block';
    modal.classList.add('show');
    document.body.classList.add('modal-open');
    var backdrop = document.createElement('div');
    backdrop.className = 'modal-backdrop fade show';
    backdrop.id = 'mcpModalBackdrop';
    document.body.appendChild(backdrop);
  }

  function hideMcpModal() {
    var modal = document.getElementById('mcpModal');
    if (!modal) return;
    modal.style.display = 'none';
    modal.classList.remove('show');
    document.body.classList.remove('modal-open');
    var backdrop = document.getElementById('mcpModalBackdrop');
    if (backdrop) backdrop.remove();
  }

  function copyText(text, btn, labelKey) {
    var orig = t(labelKey);
    var done = function () {
      btn.textContent = '✓';
      setTimeout(function () {
        btn.textContent = orig;
      }, 2000);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done).catch(function () {
        fallbackCopy(text);
        done();
      });
    } else {
      fallbackCopy(text);
      done();
    }
  }

  function fallbackCopy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
    } catch (e) {}
    document.body.removeChild(ta);
  }

  function openMcpModal() {
    fetchMcpHint(function () {
      refreshMcpConfig();
      checkAPI();
      showMcpModal();
    });
  }

  function bind() {
    injectNav();
    injectModal();
    var nav = document.getElementById('navUseMcp');
    if (nav && !nav.dataset.mcpBound) {
      nav.dataset.mcpBound = '1';
      nav.addEventListener('click', function (e) {
        e.preventDefault();
        openMcpModal();
      });
    }
    var bin = document.getElementById('mcpProfBin');
    if (bin) {
      if (!bin.value.trim() && serverProfBin()) bin.value = serverProfBin();
      try {
        var saved = localStorage.getItem(BIN_KEY);
        if (isProfPath(saved) && !bin.value.trim()) bin.value = saved;
        else if (!isProfPath(saved)) localStorage.removeItem(BIN_KEY);
      } catch (e) {}
      bin.addEventListener('input', refreshMcpConfig);
    }
    var closeBtn = document.getElementById('mcpModalClose');
    var dismissBtn = document.getElementById('mcpModalDismiss');
    var modal = document.getElementById('mcpModal');
    if (closeBtn) closeBtn.addEventListener('click', hideMcpModal);
    if (dismissBtn) dismissBtn.addEventListener('click', hideMcpModal);
    if (modal) {
      modal.addEventListener('click', function (e) {
        if (e.target === modal) hideMcpModal();
      });
    }
    var copyCfg = document.getElementById('mcpCopyConfigBtn');
    var copyArgs = document.getElementById('mcpCopyArgsBtn');
    if (copyCfg) {
      copyCfg.addEventListener('click', function () {
        copyText(buildMcpJSON(), copyCfg, 'mcp.copyConfig');
      });
    }
    if (copyArgs) {
      copyArgs.addEventListener('click', function () {
        copyText(buildArgsJSON(), copyArgs, 'mcp.copyArgs');
      });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bind);
  } else {
    bind();
  }
})();
