(function () {
  'use strict';

  var _settingsI18n = {};
  (function () {
    var lang = localStorage.getItem('prof-lang') || 'zh';
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/static/i18n/' + lang + '.json', false); // async=false intentional: local tool
    try {
      xhr.send();
      if (xhr.status === 200) _settingsI18n = JSON.parse(xhr.responseText);
    } catch (e) {}
  })();

  function st(key, fallback) {
    return _settingsI18n[key] !== undefined ? _settingsI18n[key] : fallback;
  }

  // Returns { container, mode } where mode is 'pprof' or 'navbar'
  function findHeader() {
    var pprofHeader = document.querySelector('.header');
    if (pprofHeader) return { container: pprofHeader, mode: 'pprof' };
    var navbarNav = document.querySelector('.navbar-nav');
    if (navbarNav) return { container: navbarNav, mode: 'navbar' };
    return null;
  }

  function init() {
    var found = findHeader();
    if (!found) return;

    var container = found.container;
    var mode = found.mode;

    // ── Settings button ──────────────────────────────────────────────────────
    var btn, btnInner;
    if (mode === 'navbar') {
      // Bootstrap navbar: inject as a <li class="nav-item">
      btn = document.createElement('li');
      btn.className = 'nav-item';
      btn.id = 'prof-settings-btn';
      btn.setAttribute('data-prof-plugin', 'settings');
      btn.style.cssText = 'cursor:pointer;user-select:none;';
      btnInner = document.createElement('span');
      btnInner.className = 'nav-link';
      btnInner.style.cssText = 'font-size:16px;padding-top:6px;';
      btnInner.textContent = '⚙';
      btn.appendChild(btnInner);
    } else {
      // pprof UI header: inject as a .menu-item div
      btn = document.createElement('div');
      btn.className = 'menu-item';
      btn.id = 'prof-settings-btn';
      btn.setAttribute('data-prof-plugin', 'settings');
      btn.style.cssText = 'cursor:pointer;padding:0 8px;user-select:none;';
      btn.innerHTML = '<div class="menu-name" style="font-size:16px">⚙</div>';
    }

    // ── Settings panel (hidden by default) ───────────────────────────────────
    var panel = document.createElement('div');
    panel.id = 'prof-settings-panel';
    panel.setAttribute('data-prof-plugin', 'settings');
    panel.style.cssText = [
      'display:none',
      'position:fixed',
      'top:48px',
      'right:12px',
      'background:#fff',
      'border:1px solid #ddd',
      'border-radius:4px',
      'box-shadow:0 2px 8px rgba(0,0,0,.12)',
      'padding:12px 16px',
      'min-width:220px',
      'z-index:9999',
      'font-size:13px',
      'color:#333',
      'line-height:1.4',
    ].join(';');

    fetchAndRender(panel);

    btn.addEventListener('click', function (e) {
      e.stopPropagation();
      panel.style.display = panel.style.display === 'none' ? 'block' : 'none';
    });

    document.addEventListener('click', function () {
      panel.style.display = 'none';
    });

    container.appendChild(btn);
    document.body.appendChild(panel);
  }

  // ── Fetch /api/plugins and render toggle list ────────────────────────────
  function fetchAndRender(panel) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/plugins');
    xhr.onload = function () {
      if (xhr.status !== 200) return;
      try {
        var data = JSON.parse(xhr.responseText);
        renderPanel(panel, data.plugins || []);
      } catch (e) {
        console.warn('[settings] failed to parse /api/plugins:', e);
      }
    };
    xhr.onerror = function () {
      console.warn('[settings] /api/plugins request failed');
    };
    xhr.send();
  }

  function renderPanel(panel, plugins) {
    panel.innerHTML = '<div style="font-weight:600;margin-bottom:10px;font-size:14px">' + st('settings.title', 'Prof 插件') + '</div>';

    if (plugins.length === 0) {
      var empty = document.createElement('div');
      empty.style.cssText = 'color:#999;font-size:12px;';
      empty.textContent = st('settings.noPlugins', '暂无插件');
      panel.appendChild(empty);
      return;
    }

    plugins.forEach(function (p) {
      var lsKey = 'prof-plugin-' + p.name;
      var lsVal = localStorage.getItem(lsKey);
      var isOn = lsVal !== null ? lsVal === 'true' : p.enabled;

      var row = document.createElement('div');
      row.style.cssText = 'display:flex;justify-content:space-between;align-items:center;margin:6px 0;gap:12px;';

      var label = document.createElement('span');
      label.textContent = st('plugin.' + p.name, p.description || p.name);
      label.style.cssText = 'flex:1;';

      var toggle = document.createElement('input');
      toggle.type = 'checkbox';
      toggle.checked = isOn;
      toggle.style.cssText = 'cursor:pointer;width:16px;height:16px;flex-shrink:0;';

      row.appendChild(label);
      row.appendChild(toggle);
      panel.appendChild(row);

      // 子配置区域
      if (p.name === 'source-fold') {
        var subConfig = renderSourceFoldConfig();
        subConfig.style.display = isOn ? 'block' : 'none';
        panel.appendChild(subConfig);

        toggle.addEventListener('change', function () {
          localStorage.setItem(lsKey, toggle.checked ? 'true' : 'false');
          subConfig.style.display = toggle.checked ? 'block' : 'none';
          showReloadHint(panel);
        });
      } else if (p.name === 'peek-fold') {
        var peekSubConfig = renderPeekFoldConfig();
        peekSubConfig.style.display = isOn ? 'block' : 'none';
        panel.appendChild(peekSubConfig);

        toggle.addEventListener('change', function () {
          localStorage.setItem(lsKey, toggle.checked ? 'true' : 'false');
          peekSubConfig.style.display = toggle.checked ? 'block' : 'none';
          showReloadHint(panel);
        });
      } else {
        toggle.addEventListener('change', function () {
          localStorage.setItem(lsKey, toggle.checked ? 'true' : 'false');
          showReloadHint(panel);
        });
      }
    });
  }

  // source-fold 子配置：标准库 / 第三方 / 全部
  function renderSourceFoldConfig() {
    var configKey = 'prof-source-fold-config';
    var cfg = { collapseStdlib: true, collapseThirdParty: false, collapseAll: false };
    try {
      var raw = localStorage.getItem(configKey);
      if (raw) cfg = Object.assign({ collapseStdlib: true, collapseThirdParty: false, collapseAll: false }, JSON.parse(raw));
    } catch (e) {}

    var container = document.createElement('div');
    container.style.cssText = [
      'margin:2px 0 6px 12px',
      'padding:6px 10px',
      'background:#f9f9f9',
      'border-left:3px solid #ddd',
      'font-size:12px',
    ].join(';');

    function makeSubRow(labelText, checked, onChange) {
      var row = document.createElement('label');
      row.style.cssText = 'display:flex;align-items:center;gap:6px;margin:3px 0;cursor:pointer;';
      var cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.checked = checked;
      cb.style.cssText = 'cursor:pointer;';
      cb.addEventListener('change', function () { onChange(cb.checked); });
      var txt = document.createElement('span');
      txt.textContent = labelText;
      row.appendChild(cb);
      row.appendChild(txt);
      return row;
    }

    function saveConfig() {
      localStorage.setItem(configKey, JSON.stringify(cfg));
    }

    container.appendChild(makeSubRow(st('source-fold.collapseStdlib', '折叠 std（标准库）'), cfg.collapseStdlib, function (v) {
      cfg.collapseStdlib = v;
      saveConfig();
    }));
    container.appendChild(makeSubRow(st('source-fold.collapseThirdParty', '折叠 mod（第三方）'), cfg.collapseThirdParty, function (v) {
      cfg.collapseThirdParty = v;
      saveConfig();
    }));
    container.appendChild(makeSubRow(st('source-fold.collapseAll', '折叠全部'), cfg.collapseAll, function (v) {
      cfg.collapseAll = v;
      saveConfig();
    }));

    return container;
  }

  // peek-fold 子配置：折叠 std/mod/全部 + 路径缩写
  function renderPeekFoldConfig() {
    var configKey = 'prof-peek-fold-config';
    var cfg = { collapseStd: true, collapseMod: false, collapseAll: false, shortenPaths: true };
    try {
      var raw = localStorage.getItem(configKey);
      if (raw) cfg = Object.assign({ collapseStd: true, collapseMod: false, collapseAll: false, shortenPaths: true }, JSON.parse(raw));
    } catch (e) {}

    var container = document.createElement('div');
    container.style.cssText = [
      'margin:2px 0 6px 12px',
      'padding:6px 10px',
      'background:#f9f9f9',
      'border-left:3px solid #ddd',
      'font-size:12px',
    ].join(';');

    function makeSubRow(labelText, checked, onChange) {
      var row = document.createElement('label');
      row.style.cssText = 'display:flex;align-items:center;gap:6px;margin:3px 0;cursor:pointer;';
      var cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.checked = checked;
      cb.style.cssText = 'cursor:pointer;';
      cb.addEventListener('change', function () { onChange(cb.checked); });
      var txt = document.createElement('span');
      txt.textContent = labelText;
      row.appendChild(cb);
      row.appendChild(txt);
      return row;
    }

    function saveConfig() {
      localStorage.setItem(configKey, JSON.stringify(cfg));
    }

    container.appendChild(makeSubRow(st('peek-fold.collapseStd', '折叠 std（标准库）'), cfg.collapseStd, function (v) {
      cfg.collapseStd = v;
      saveConfig();
    }));
    container.appendChild(makeSubRow(st('peek-fold.collapseMod', '折叠 mod（第三方）'), cfg.collapseMod, function (v) {
      cfg.collapseMod = v;
      saveConfig();
    }));
    container.appendChild(makeSubRow(st('peek-fold.collapseAll', '折叠全部'), cfg.collapseAll, function (v) {
      cfg.collapseAll = v;
      saveConfig();
    }));
    container.appendChild(makeSubRow(st('peek-fold.shortenPaths', '缩短路径显示（std/ mod/）'), cfg.shortenPaths, function (v) {
      cfg.shortenPaths = v;
      saveConfig();
    }));

    return container;
  }

  function showReloadHint(panel) {
    if (panel.querySelector('#prof-reload-hint')) return;
    var hint = document.createElement('div');
    hint.id = 'prof-reload-hint';
    hint.style.cssText = [
      'margin-top:10px',
      'padding-top:8px',
      'border-top:1px solid #eee',
      'color:#1565c0',
      'font-size:12px',
      'cursor:pointer',
      'text-align:center',
    ].join(';');
    hint.textContent = st('settings.reload', '点击刷新生效');
    hint.addEventListener('click', function () { location.reload(); });
    panel.appendChild(hint);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
