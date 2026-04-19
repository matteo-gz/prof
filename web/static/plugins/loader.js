(function () {
  'use strict';

  var MANIFEST = [
    '/static/plugins/i18n-zh.js',
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

  function loadScripts() {
    if (MANIFEST.length === 0) {
      onAllLoaded();
      return;
    }
    var loaded = 0;
    for (var i = 0; i < MANIFEST.length; i++) {
      var s = document.createElement('script');
      s.src = MANIFEST[i];
      s.async = false;
      s.onload = s.onerror = function () {
        loaded++;
        if (loaded >= MANIFEST.length) {
          onAllLoaded();
        }
      };
      document.head.appendChild(s);
    }
  }

  function onAllLoaded() {
    runtime._initAll();
    startMutationObserver();
    startCharacterDataObservers();
    startViewMonitor();
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
    document.addEventListener('DOMContentLoaded', loadScripts);
  } else {
    loadScripts();
  }
})();
