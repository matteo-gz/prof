(function (P) {
  'use strict';

  var SEPARATOR_RE = /^[-]{20,}\+[-]+/;

  // 从 localStorage 读取配置
  function loadConfig() {
    var defaults = { collapseStd: true, collapseMod: false, collapseAll: false, shortenPaths: true };
    try {
      var raw = localStorage.getItem('prof-peek-fold-config');
      if (raw) return Object.assign(defaults, JSON.parse(raw));
    } catch (e) {}
    return defaults;
  }

  // 判断是否是主节点行（| 后1个空格，不是3个空格）
  function isNodeLine(line) {
    return /\| \S/.test(line) && !/\|\s{3}/.test(line.replace('| ', '|X'));
  }

  // 从主节点行提取函数名
  function extractNodeName(line) {
    var m = line.match(/\|\s(.+)$/);
    return m ? m[1].trim() : line.trim();
  }

  // 根据 block 的 bodyLines 中的文件路径分类
  function classifyBlock(block, roots) {
    for (var i = 0; i < block.bodyLines.length; i++) {
      var m = block.bodyLines[i].match(/(\/[^\s]+\.go:\d+)/);
      if (!m) continue;
      var fp = m[1];
      if (roots.mod && fp.indexOf(roots.mod) === 0) return 'thirdparty';
      if (roots.std && fp.indexOf(roots.std) === 0) return 'stdlib';
      if (fp.indexOf('/pkg/mod/') !== -1) return 'thirdparty';
      return 'local';
    }
    // 兜底：包名无域名 → stdlib
    var pkg = extractPkg(block.nodeFunc);
    return (pkg && pkg.split('/')[0].indexOf('.') === -1) ? 'stdlib' : 'local';
  }

  // 判断是否默认折叠
  function shouldFold(block, roots, config) {
    if (config.collapseAll) return true;
    var category = classifyBlock(block, roots);
    if (config.collapseStd && category === 'stdlib') return true;
    if (config.collapseMod && category === 'thirdparty') return true;
    return false;
  }

  // ── 路径缩写模块 ──────────────────────────────────────────────────────────

  // "runtime.semasleep" → "runtime"
  // "github.com/gin-gonic/gin.(*C).Next" → "github.com/gin-gonic/gin"
  function extractPkg(funcName) {
    return funcName
      .replace(/\(\*?[^)]+\)\.[^.]+$/, '')  // 去掉 (*Type).Method
      .replace(/\.[^./]+$/, '');             // 去掉最后 .FuncName
  }

  // 扫描所有行，发现 std/mod 路径根
  // peek 行格式：" ... | funcname /abs/path/file.go:N"，行首是空格+数字，不能用 ^
  function discoverRoots(lines) {
    var roots = { std: null, mod: null };
    for (var i = 0; i < lines.length; i++) {
      var m = lines[i].match(/\|\s+(\S+)\s+(\/[^\s]+\.go:\d+)/);
      if (!m) continue;
      var funcName = m[1];
      var filePath = m[2];
      var modIdx = filePath.indexOf('/pkg/mod/');
      if (modIdx !== -1 && !roots.mod) {
        roots.mod = filePath.slice(0, modIdx + '/pkg/mod/'.length);
      }
      if (!roots.std && modIdx === -1) {
        var pkg = extractPkg(funcName);
        if (pkg && pkg.indexOf('.') === -1) {
          var pkgIdx = filePath.indexOf('/' + pkg + '/');
          if (pkgIdx !== -1) {
            roots.std = filePath.slice(0, pkgIdx + 1);
          }
        }
      }
      if (roots.std && roots.mod) break;
    }
    return roots;
  }

  function shortenPath(filePath, roots) {
    if (roots.mod && filePath.indexOf(roots.mod) === 0) {
      return 'mod/' + filePath.slice(roots.mod.length);
    }
    if (roots.std && filePath.indexOf(roots.std) === 0) {
      return 'std/' + filePath.slice(roots.std.length);
    }
    return filePath;
  }

  // 对单行做路径缩写替换（只替换 /abs/path/file.go:N 形式）
  function shortenLine(line, roots) {
    return line.replace(/(\/[^\s]+\.go:\d+)/g, function (path) {
      return shortenPath(path, roots);
    });
  }

  // 解析 pre 文本，返回 { header, blocks: [{nodeFunc, bodyLines}], roots }
  function parseBlocks(text) {
    var lines = text.split('\n');
    var blocks = [];
    var current = null;
    var headerLines = [];
    var inHeader = true;

    for (var i = 0; i < lines.length; i++) {
      var line = lines[i];
      if (SEPARATOR_RE.test(line)) {
        if (current !== null) {
          blocks.push(current);
          current = null;
        }
        // legend 行之前的 separator：头部
        if (inHeader && i + 1 < lines.length && lines[i + 1].indexOf('flat') !== -1) {
          inHeader = false;
          headerLines.push(line);
          continue;
        }
        if (inHeader) {
          headerLines.push(line);
          continue;
        }
        // 开始新块
        current = { nodeFunc: '', bodyLines: [] };
        continue;
      }
      if (inHeader) {
        headerLines.push(line);
        continue;
      }
      if (current === null) continue;

      if (isNodeLine(line)) {
        current.nodeFunc = extractNodeName(line);
      }
      current.bodyLines.push(line);
    }

    var roots = discoverRoots(lines);
    return { header: headerLines.join('\n'), blocks: blocks, roots: roots };
  }

  // 构建折叠块 DOM（保持等宽字体格式）
  function buildBlock(nodeFunc, bodyLines, folded, roots, config) {
    var wrapper = document.createElement('div');
    wrapper.setAttribute('data-prof-plugin', 'peek-fold');
    wrapper.style.cssText = 'margin:2px 0;font-family:monospace;font-size:13px;';

    var title = document.createElement('div');
    title.setAttribute('data-prof-plugin', 'peek-fold');
    title.setAttribute('data-folded', folded ? 'true' : 'false');
    title.style.cssText = [
      'cursor:pointer',
      'padding:1px 4px',
      'background:#f5f5f5',
      'border-left:3px solid #ccc',
      'white-space:pre',
      'overflow:hidden',
      'text-overflow:ellipsis',
    ].join(';');

    var indicator = document.createElement('span');
    indicator.className = 'prof-peek-indicator';
    indicator.style.cssText = 'margin-right:6px;color:#888;font-style:normal;';
    indicator.textContent = folded ? '▶' : '▼';

    var funcLabel = document.createElement('span');
    var enableShorten = config.shortenPaths !== false && roots;
    var displayFunc = enableShorten ? shortenLine(nodeFunc || '', roots) : (nodeFunc || '');
    funcLabel.textContent = displayFunc || '(unknown)';

    title.appendChild(indicator);
    title.appendChild(funcLabel);

    var body = document.createElement('pre');
    body.setAttribute('data-prof-plugin', 'peek-fold');
    body.style.cssText = [
      'margin:0',
      'padding:2px 4px 2px 16px',
      'border-left:3px solid #eee',
      'background:#fafafa',
      'display:' + (folded ? 'none' : 'block'),
      'font-family:monospace',
      'font-size:13px',
    ].join(';');
    var displayLines = enableShorten
      ? bodyLines.map(function (l) { return shortenLine(l, roots); })
      : bodyLines;
    body.textContent = displayLines.join('\n');

    title.addEventListener('click', function () {
      var isFolded = title.getAttribute('data-folded') === 'true';
      var next = !isFolded;
      title.setAttribute('data-folded', next ? 'true' : 'false');
      body.style.display = next ? 'none' : 'block';
      indicator.textContent = next ? '▶' : '▼';
    });

    wrapper.appendChild(title);
    wrapper.appendChild(body);
    return wrapper;
  }

  // 注入工具栏（全部折叠 / 全部展开）
  function injectToolbar(container, titleNodes) {
    var toolbar = document.createElement('div');
    toolbar.setAttribute('data-prof-plugin', 'peek-fold');
    toolbar.style.cssText = 'padding:4px 0 8px 0;display:flex;gap:8px;';

    function makeBtn(text, action) {
      var btn = document.createElement('button');
      btn.textContent = text;
      btn.setAttribute('data-prof-plugin', 'peek-fold');
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
      titleNodes.forEach(function (t) {
        t.setAttribute('data-folded', 'true');
        t.querySelector('.prof-peek-indicator').textContent = '▶';
        t.nextElementSibling.style.display = 'none';
      });
    }));
    toolbar.appendChild(makeBtn('全部展开', function () {
      titleNodes.forEach(function (t) {
        t.setAttribute('data-folded', 'false');
        t.querySelector('.prof-peek-indicator').textContent = '▼';
        t.nextElementSibling.style.display = 'block';
      });
    }));
    container.insertBefore(toolbar, container.firstChild);
    return toolbar;
  }

  // 清理上次注入
  function cleanup(content) {
    content.querySelectorAll('[data-prof-plugin="peek-fold"]').forEach(function (el) {
      el.parentNode && el.parentNode.removeChild(el);
    });
    var pre = content.querySelector('pre');
    if (pre) pre.style.display = '';
  }

  function run() {
    var content = document.getElementById('content');
    if (!content) return;
    var pre = content.querySelector('pre');
    if (!pre) return;
    // 只在 peek 视图运行
    if (window.location.pathname.indexOf('/peek') === -1) return;

    cleanup(content);

    var text = pre.textContent;
    var parsed = parseBlocks(text);
    if (parsed.blocks.length === 0) return;

    var config = loadConfig();
    pre.style.display = 'none';

    // 头部元信息
    var headerPre = document.createElement('pre');
    headerPre.setAttribute('data-prof-plugin', 'peek-fold');
    headerPre.style.cssText = 'margin:0;padding:4px;font-size:13px;font-family:monospace;color:#555;';
    headerPre.textContent = parsed.header;
    content.insertBefore(headerPre, pre);

    // 各函数块
    var titleNodes = [];
    parsed.blocks.forEach(function (block) {
      var folded = shouldFold(block, parsed.roots, config);
      var wrapper = buildBlock(block.nodeFunc, block.bodyLines, folded, parsed.roots, config);
      content.insertBefore(wrapper, pre);
      var titleEl = wrapper.querySelector('[data-folded]');
      if (titleEl) titleNodes.push(titleEl);
    });

    injectToolbar(content, titleNodes);
  }

  P.register({
    name: 'peek-fold',
    version: '1.0.0',
    views: ['peek'],

    init: function () { run(); },
    onViewChange: function (view) {
      if (view === 'peek') run();
    },
  });

})(window.ProfPlugin);
