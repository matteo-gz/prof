(function () {
  'use strict';

  // MANIFEST: plugin name → static src path; all default-off via /api/plugins
  var MANIFEST = [
    { name: 'i18n-zh',       src: '/static/plugins/i18n-zh.js' },
    { name: 'source-fold',   src: '/static/plugins/source-fold.js' },
    { name: 'peek-fold',     src: '/static/plugins/peek-fold.js' },
    { name: 'graph-explain', src: '/static/plugins/graph-explain.js' },
  ];

  var runtime = {
    plugins: [],
    shared: {},
    _loaded: 0,

    register: function (plugin) {
      if (!plugin.name) return;
      plugin.views = plugin.views || ['*'];
      this.plugins.push(plugin);
    },

    _currentView: function () {
      var path = window.location.pathname;
      if (path.indexOf('/flamegraph') !== -1) return 'flamegraph';
      if (path.indexOf('/top') !== -1) return 'top';
      if (path.indexOf('/peek') !== -1) return 'peek';
      if (path.indexOf('/source') !== -1) return 'source';
      if (path.indexOf('/disasm') !== -1) return 'disasm';
      return 'graph';
    },

    _shouldRun: function (plugin, view) {
      if (!plugin.views) return true;
      if (plugin.views.indexOf('*') !== -1) return true;
      return plugin.views.indexOf(view) !== -1;
    },

    _initAll: function () {
      var view = this._currentView();
      for (var i = 0; i < this.plugins.length; i++) {
        var p = this.plugins[i];
        if (p.init && this._shouldRun(p, view)) {
          try { p.init(); } catch (e) {
            console.error('[ProfPlugin] init error in ' + p.name + ':', e);
          }
        }
      }
    },

    _notifyViewChange: function (view) {
      for (var i = 0; i < this.plugins.length; i++) {
        var p = this.plugins[i];
        if (p.onViewChange && this._shouldRun(p, view)) {
          try { p.onViewChange(view); } catch (e) {
            console.error('[ProfPlugin] onViewChange error in ' + p.name + ':', e);
          }
        }
      }
    },

    _notifyMutation: function (nodes) {
      var view = this._currentView();
      for (var i = 0; i < this.plugins.length; i++) {
        var p = this.plugins[i];
        if (p.onMutation && this._shouldRun(p, view)) {
          try { p.onMutation(nodes); } catch (e) {
            console.error('[ProfPlugin] onMutation error in ' + p.name + ':', e);
          }
        }
      }
    },

    _notifyCharacterData: function (target) {
      var view = this._currentView();
      for (var i = 0; i < this.plugins.length; i++) {
        var p = this.plugins[i];
        if (p.onCharacterData && this._shouldRun(p, view)) {
          try { p.onCharacterData(target); } catch (e) {
            console.error('[ProfPlugin] onCharacterData error in ' + p.name + ':', e);
          }
        }
      }
    }
  };

  window.ProfPlugin = runtime;

  // ── Fetch server-side plugin defaults ──────────────────────────────────────
  function fetchPluginConfig(callback) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/plugins');
    xhr.onload = function () {
      var config = {};
      if (xhr.status === 200) {
        try {
          var data = JSON.parse(xhr.responseText);
          (data.plugins || []).forEach(function (p) {
            config[p.name] = p.enabled;
          });
        } catch (e) {
          console.warn('[ProfPlugin] failed to parse /api/plugins:', e);
        }
      }
      callback(config);
    };
    xhr.onerror = function () { callback({}); };
    xhr.send();
  }

  // ── Merge localStorage overrides (highest priority) ────────────────────────
  function resolveEnabled(serverConfig) {
    var result = {};
    Object.keys(serverConfig).forEach(function (name) {
      var lsKey = 'prof-plugin-' + name;
      var lsVal = localStorage.getItem(lsKey);
      if (lsVal !== null) {
        result[name] = lsVal === 'true';
      } else {
        result[name] = serverConfig[name];
      }
    });
    return result;
  }

  // ── Load a single script, invoke cb when done ──────────────────────────────
  function loadScript(src, cb) {
    var s = document.createElement('script');
    s.src = src;
    s.async = false;
    s.onload = s.onerror = function () { cb(); };
    document.head.appendChild(s);
  }

  // ── Load enabled plugins from MANIFEST sequentially ───────────────────────
  function loadEnabledPlugins(enabledMap, cb) {
    var toLoad = MANIFEST.filter(function (item) {
      return enabledMap[item.name] === true;
    });
    if (toLoad.length === 0) { cb(); return; }
    var loaded = 0;
    for (var i = 0; i < toLoad.length; i++) {
      loadScript(toLoad[i].src, function () {
        loaded++;
        if (loaded >= toLoad.length) { cb(); }
      });
    }
  }

  function onAllLoaded() {
    runtime._initAll();
    startMutationObserver();
    startCharacterDataObservers();
    startViewMonitor();
  }

  // ── Boot sequence ──────────────────────────────────────────────────────────
  // 1. Load settings.js always (settings panel is not governed by MANIFEST)
  // 2. Fetch /api/plugins for server defaults
  // 3. Merge localStorage overrides
  // 4. Load only the enabled plugins
  // 5. Run onAllLoaded
  function boot() {
    loadScript('/static/plugins/settings.js', function () {
      fetchPluginConfig(function (serverConfig) {
        var enabled = resolveEnabled(serverConfig);
        loadEnabledPlugins(enabled, function () {
          onAllLoaded();
        });
      });
    });
  }

  function startCharacterDataObservers() {
    if (typeof MutationObserver === 'undefined') return;
    var selectors = [];
    for (var i = 0; i < runtime.plugins.length; i++) {
      var p = runtime.plugins[i];
      if (p.characterDataSelectors) {
        selectors = selectors.concat(p.characterDataSelectors);
      }
    }
    var seen = {};
    for (var j = 0; j < selectors.length; j++) {
      var sel = selectors[j];
      if (seen[sel]) continue;
      seen[sel] = true;
      var el = document.querySelector(sel);
      if (!el) continue;
      (function (target) {
        var obs = new MutationObserver(function () {
          runtime._notifyCharacterData(target);
        });
        obs.observe(target, { characterData: true, subtree: true });
      })(el);
    }
  }

  function startMutationObserver() {
    if (typeof MutationObserver === 'undefined') return;
    var observer = new MutationObserver(function (mutations) {
      var nodes = [];
      for (var i = 0; i < mutations.length; i++) {
        for (var j = 0; j < mutations[i].addedNodes.length; j++) {
          var n = mutations[i].addedNodes[j];
          if (n.nodeType === 1) nodes.push(n);
        }
      }
      if (nodes.length > 0) {
        runtime._notifyMutation(nodes);
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });
  }

  function startViewMonitor() {
    var lastView = runtime._currentView();
    function check() {
      var view = runtime._currentView();
      if (view !== lastView) {
        lastView = view;
        runtime._notifyViewChange(view);
      }
    }
    window.addEventListener('popstate', check);
    window.addEventListener('hashchange', check);
    setInterval(check, 500);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
