(function (P) {
  'use strict';

  // 从 localStorage 读取默认折叠配置
  function loadConfig() {
    var defaults = { collapseStdlib: true, collapseThirdParty: false, collapseAll: false };
    try {
      var raw = localStorage.getItem('prof-source-fold-config');
      if (raw) return Object.assign(defaults, JSON.parse(raw));
    } catch (e) {}
    return defaults;
  }

  // 扫描所有 <p class="filename"> 发现根目录
  function discoverSourceRoots(content) {
    var roots = { std: null, mod: null };
    var ps = content.querySelectorAll('p.filename');
    for (var i = 0; i < ps.length; i++) {
      var filePath = ps[i].textContent.trim();
      var h2 = ps[i].previousElementSibling;
      var funcName = h2 ? h2.textContent.trim() : '';

      var modIdx = filePath.indexOf('/pkg/mod/');
      if (modIdx !== -1 && !roots.mod) {
        roots.mod = filePath.slice(0, modIdx + '/pkg/mod/'.length);
      }
      if (!roots.std && modIdx === -1) {
        var pkg = funcName.replace(/\(\*?[^)]+\)\.[^.]+$/, '').replace(/\.[^./]+$/, '');
        var firstPart = pkg.split('/')[0];
        if (pkg && firstPart.indexOf('.') === -1) {
          var pkgIdx = filePath.indexOf('/' + firstPart + '/');
          if (pkgIdx !== -1) roots.std = filePath.slice(0, pkgIdx + 1);
        }
      }
      if (roots.std && roots.mod) break;
    }
    return roots;
  }

  // 根据路径分类：stdlib / thirdparty / local / unknown
  function classifyPath(filePath, roots) {
    if (!filePath) return 'unknown';
    if (roots.mod && filePath.startsWith(roots.mod)) return 'thirdparty';
    if (roots.std && filePath.startsWith(roots.std)) return 'stdlib';
    return filePath.indexOf('/pkg/mod/') !== -1 ? 'thirdparty' : 'local';
  }

  // 找 h2 后面的 p.filename + pre（折叠体，连续兄弟节点）
  function getFoldBody(h2) {
    var nodes = [];
    var el = h2.nextElementSibling;
    while (el && (el.tagName === 'P' || el.tagName === 'PRE')) {
      nodes.push(el);
      el = el.nextElementSibling;
    }
    return nodes;
  }

  // 判断是否默认折叠
  function shouldDefaultFold(name, filePath, roots, config) {
    if (config.collapseAll) return true;
    var category = classifyPath(filePath, roots);
    if (config.collapseStdlib && category === 'stdlib') return true;
    if (config.collapseThirdParty && category === 'thirdparty') return true;
    return false;
  }

  function isFolded(h2) {
    return h2.getAttribute('data-folded') === 'true';
  }

  // 切换折叠状态（不 wrap DOM，用 display 控制）
  function setFolded(h2, folded) {
    var body = getFoldBody(h2);
    body.forEach(function (el) {
      el.style.display = folded ? 'none' : '';
    });
    h2.setAttribute('data-folded', folded ? 'true' : 'false');
    var indicator = h2.querySelector('.prof-fold-indicator');
    if (indicator) indicator.textContent = folded ? '▶' : '▼';
  }

  // 给 h2 加折叠指示器和点击事件
  function decorateH2(h2, folded) {
    h2.setAttribute('data-prof-plugin', 'source-fold');
    h2.setAttribute('data-folded', folded ? 'true' : 'false');
    h2.style.cursor = 'pointer';

    var indicator = document.createElement('span');
    indicator.className = 'prof-fold-indicator';
    indicator.setAttribute('data-prof-plugin', 'source-fold');
    indicator.style.cssText = 'margin-right:8px;font-size:12px;color:#888;user-select:none;';
    indicator.textContent = folded ? '▶' : '▼';
    h2.insertBefore(indicator, h2.firstChild);

    h2.addEventListener('click', function () {
      setFolded(h2, !isFolded(h2));
    });

    // 应用初始折叠状态
    if (folded) {
      getFoldBody(h2).forEach(function (el) {
        el.style.display = 'none';
      });
    }
  }

  // 注入「全部折叠」「全部展开」工具栏
  function injectToolbar(content, h2List) {
    var toolbar = document.createElement('div');
    toolbar.setAttribute('data-prof-plugin', 'source-fold');
    toolbar.style.cssText = 'padding:6px 0 10px 0;display:flex;gap:8px;';

    function makeBtn(text, action) {
      var btn = document.createElement('button');
      btn.textContent = text;
      btn.setAttribute('data-prof-plugin', 'source-fold');
      btn.style.cssText = [
        'padding:2px 10px',
        'font-size:12px',
        'cursor:pointer',
        'border:1px solid #ccc',
        'border-radius:3px',
        'background:#f5f5f5',
      ].join(';');
      btn.addEventListener('click', action);
      return btn;
    }

    toolbar.appendChild(makeBtn('全部折叠', function () {
      h2List.forEach(function (h2) { setFolded(h2, true); });
    }));
    toolbar.appendChild(makeBtn('全部展开', function () {
      h2List.forEach(function (h2) { setFolded(h2, false); });
    }));

    content.insertBefore(toolbar, content.firstChild);
  }

  // 防重复：清理上次注入的元素
  function cleanup(content) {
    var old = content.querySelectorAll('[data-prof-plugin="source-fold"]');
    old.forEach(function (el) {
      // toolbar div 整体移除，指示器 span 移除，h2 属性还原
      if (el.tagName === 'DIV' || el.tagName === 'SPAN') {
        el.parentNode && el.parentNode.removeChild(el);
      } else if (el.tagName === 'H2') {
        el.removeAttribute('data-prof-plugin');
        el.removeAttribute('data-folded');
        el.style.cursor = '';
      }
    });
    // 恢复隐藏的 p/pre
    var bodies = content.querySelectorAll('p[data-prof-fold-body], pre[data-prof-fold-body]');
    bodies.forEach(function (el) {
      el.style.display = '';
      el.removeAttribute('data-prof-fold-body');
    });
  }

  function run() {
    var content = document.getElementById('content');
    if (!content || !content.classList.contains('source')) return;

    var h2List = Array.prototype.slice.call(content.querySelectorAll('h2'));
    if (h2List.length === 0) return;

    cleanup(content);

    var config = loadConfig();
    var roots = discoverSourceRoots(content);

    h2List.forEach(function (h2) {
      var p = h2.nextElementSibling;
      var filePath = (p && p.tagName === 'P') ? p.textContent.trim() : '';
      var name = h2.textContent.trim();
      var folded = shouldDefaultFold(name, filePath, roots, config);
      decorateH2(h2, folded);
    });

    injectToolbar(content, h2List);
  }

  P.register({
    name: 'source-fold',
    version: '1.1.0',
    views: ['source'],

    init: function () { run(); },
    onViewChange: function (view) {
      if (view === 'source') run();
    },
  });

})(window.ProfPlugin);
