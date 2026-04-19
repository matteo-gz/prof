(function (P) {
  'use strict';

  var panel = null;  // 全局唯一面板

  // 从 URL ?si= 读取 profile 类型，返回 { key, label, desc }
  var SI_LABELS = {
    'alloc_space':   { label: 'alloc_space',   desc: '历史累计分配字节数' },
    'alloc_objects': { label: 'alloc_objects',  desc: '历史累计分配对象数' },
    'inuse_space':   { label: 'inuse_space',    desc: '当前使用中字节数' },
    'inuse_objects': { label: 'inuse_objects',  desc: '当前使用中对象数' },
    'cpu':           { label: 'cpu',            desc: 'CPU 占用时间' },
    'contentions':   { label: 'contentions',    desc: '锁竞争次数' },
    'delay':         { label: 'delay',          desc: '锁等待时间' },
  };
  function getProfileType() {
    try {
      var si = new URLSearchParams(window.location.search).get('si') || '';
      return SI_LABELS[si] || (si ? { label: si, desc: '' } : null);
    } catch (e) { return null; }
  }

  // SVG id "node3" → DOT id "N3"
  function toDotId(svgId) {
    return 'N' + svgId.replace('node', '');
  }

  // dotId "N3" → 从节点 <a xlink:title> 取函数名
  function dotIdToFunc(graph0, id) {
    var svgId = 'node' + id.replace('N', '');
    var g = graph0.querySelector('#' + svgId);
    if (!g) return id;
    var a = g.querySelector('a');
    var t = a ? (a.getAttribute('xlink:title') || '') : '';
    var m = t.match(/^(.+?)\s*\(/);
    return m ? m[1].trim() : id;
  }

  // 从当前节点向上追溯，沿 cum 最大的父节点走，最多 maxHops 步
  // 返回 [{dotId, func}, ...] 从 root 到 current
  function buildCallChain(graph0, dotId, maxHops) {
    maxHops = maxHops || 6;

    function findCallerIds(id) {
      var result = [];
      graph0.querySelectorAll('g.edge').forEach(function (e) {
        var titleEl = e.querySelector('title');
        if (!titleEl) return;
        var txt = titleEl.textContent.trim();
        if (/NN\d+/.test(txt)) return;
        var parts = txt.split('->');
        if (parts.length !== 2) return;
        if (parts[1].trim() === id) result.push(parts[0].trim());
      });
      return result;
    }

    function getNodeCum(id) {
      var svgId = 'node' + id.replace('N', '');
      var g = graph0.querySelector('#' + svgId);
      if (!g) return 0;
      var a = g.querySelector('a');
      var t = a ? (a.getAttribute('xlink:title') || '') : '';
      var m = t.match(/([\d.]+)(kB|MB|B)/i);
      return m ? parseBytes(m[1] + m[2]) : 0;
    }

    var chain = [{ dotId: dotId, func: dotIdToFunc(graph0, dotId) }];
    var current = dotId;
    var visited = {}; visited[dotId] = true;

    for (var i = 0; i < maxHops; i++) {
      var callers = findCallerIds(current).filter(function (c) { return !visited[c]; });
      if (callers.length === 0) break;
      // 选 cum 最大的父节点（热路径）
      var best = callers[0], bestCum = getNodeCum(callers[0]);
      callers.forEach(function (c) {
        var v = getNodeCum(c);
        if (v > bestCum) { bestCum = v; best = c; }
      });
      visited[best] = true;
      chain.unshift({ dotId: best, func: dotIdToFunc(graph0, best) });
      current = best;
    }
    return chain;
  }

  // 构造 pprof Source 视图 URL
  function buildSourceUrl(funcName) {
    try {
      var base = window.location.pathname.replace(/\/[^/]+$/, '');
      var si   = new URLSearchParams(window.location.search).get('si') || '';
      var esc  = funcName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      return base + '/source?f=' + encodeURIComponent(esc) + (si ? '&si=' + si : '');
    } catch (e) { return ''; }
  }

  // 在图中按 cum 排名（仅统计 node* 节点，不含 nodelet）
  function getNodeRanking(graph0, dotId) {
    var nodes = [];
    graph0.querySelectorAll('g.node').forEach(function (g) {
      if (!g.id || !/^node\d+$/.test(g.id)) return;
      var a = g.querySelector('a');
      var t = a ? (a.getAttribute('xlink:title') || '') : '';
      var m = t.match(/([\d.]+)(kB|MB|B)/i);
      if (!m) return;
      nodes.push({ id: g.id, cum: parseBytes(m[1] + m[2]) });
    });
    nodes.sort(function (a, b) { return b.cum - a.cum; });
    var myId = 'node' + dotId.replace('N', '');
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i].id === myId) return { rank: i + 1, total: nodes.length };
    }
    return null;
  }

  // 从函数全名提取包路径并归类：标准库 / 业务代码 / 第三方库
  function getPackageInfo(funcName) {
    // 去掉方法名部分，提取包路径
    var pkg = funcName.replace(/\.\([^)]*\).*$/, '').replace(/\.[^./]+$/, '');
    var firstSeg = pkg.split('/')[0];
    if (!firstSeg.includes('.')) {
      return { pkg: pkg, badge: '标准库',  color: '#1b5e20', bg: '#e8f5e9' };
    }
    if (pkg.indexOf('/internal/') !== -1 || pkg.indexOf('/cmd/') !== -1) {
      return { pkg: pkg, badge: '业务代码', color: '#bf360c', bg: '#fbe9e7' };
    }
    return { pkg: pkg, badge: '第三方库', color: '#4a148c', bg: '#f3e5f5' };
  }

  // 从 <a xlink:title="funcname (cum)"> 提取函数名和 cum
  function parseNodeTitle(g) {
    var a = g.querySelector('a');
    var t = a ? (a.getAttribute('xlink:title') || '') : '';
    // "biz.(*Usecase).Proxy.func2 (5926.07kB)"
    var m = t.match(/^(.+?)\s*\(([^)]+)\)$/);
    return {
      name: m ? m[1].trim() : t.trim(),
      cum: m ? m[2] : '',
    };
  }

  // 从 <text> 元素提取 flat（自身占比行）和 cum（"of ..." 行带百分比）
  function parseNodeTexts(g) {
    var flat = '', cum = '';
    g.querySelectorAll('text').forEach(function (tx) {
      var s = tx.textContent.trim();
      // flat 行：包含 (X%) 且不以 "of " 开头
      if (!flat && /\([\d.]+%\)/.test(s) && s.indexOf('of ') !== 0) {
        flat = s;
      }
      // cum 行：以 "of " 开头且包含 (X%)
      if (!cum && s.indexOf('of ') === 0 && /\([\d.]+%\)/.test(s)) {
        cum = s;
      }
    });
    return { flat: flat, cum: cum };
  }

  // 用 DOT ID 在 graph0 内查找进出边
  function findEdges(graph0, dotId) {
    var ins = [], outs = [], inlines = [];
    graph0.querySelectorAll('g.edge').forEach(function (e) {
      var titleEl = e.querySelector('title');
      if (!titleEl) return;
      var titleText = titleEl.textContent.trim(); // "N1->N7" 或 "N3->NN3_0"

      // nodelet 出边（N3->NN3_0）：收集为内联分配，仅处理属于本节点的
      var nodeletM = titleText.match(/^(.+)->NN(\d+_\d+)$/);
      if (nodeletM) {
        if (nodeletM[1] === dotId) {
          var a = e.querySelector('a');
          var weight = a ? (a.getAttribute('xlink:title') || '') : '';
          // 从 nodelet 节点的 <text> 取尺寸标签（如 "64kB..72kB"）
          var nodeletId = 'NN' + nodeletM[2];
          var nodeletG = graph0.querySelector('#' + nodeletId);
          var sizeLabel = '';
          if (nodeletG) {
            var tx = nodeletG.querySelector('text');
            if (tx) sizeLabel = tx.textContent.trim();
          }
          inlines.push({ label: sizeLabel || nodeletId, weight: weight });
        }
        return;
      }

      // 跳过其他含 NN 的边（防御）
      if (/NN\d+/.test(titleText)) return;

      // 精确匹配：split('->')，避免 N3 误匹配 N31/N32 等
      var parts = titleText.split('->');
      if (parts.length !== 2) return;
      var src = parts[0].trim();
      var dst = parts[1].trim();
      var isOut = src === dotId;
      var isIn  = dst === dotId;
      if (!isOut && !isIn) return;
      // xlink:title 格式："srcFunc -> destFunc (weight)" 或 "srcFunc ... destFunc (weight)"
      var a = e.querySelector('a');
      var xl = a ? (a.getAttribute('xlink:title') || '') : '';
      var m = xl.match(/^(.+?)\s*(?:->|\.\.\.)\s*(.+?)\s*\(([^)]+)\)$/);
      if (!m) return;
      if (isOut) outs.push({ func: m[2].trim(), weight: m[3] });
      if (isIn)  ins.push({ func: m[1].trim(), weight: m[3] });
    });
    return { ins: ins, outs: outs, inlines: inlines };
  }

  // 根据 Flat/Cum 百分比比值生成诊断文字
  function diagnoseFlatCum(flatText, cumText) {
    var fm = (flatText || '').match(/([\d.]+)%/);
    var cm = (cumText || '').match(/([\d.]+)%/);
    if (!cm) return '';
    var flatPct = fm ? parseFloat(fm[1]) : 0;
    var cumPct  = parseFloat(cm[1]);
    var ratio   = cumPct > 0 ? flatPct / cumPct : 0;
    if (flatPct === 0) {
      return 'Flat = 0，纯调用入口，自身不分配，热点在下游子节点';
    }
    if (ratio > 0.8) {
      return 'Flat/Cum 比值高（' + Math.round(ratio * 100) + '%），直接分配热点，优化此函数本身';
    }
    if (ratio > 0.3) {
      return 'Flat/Cum 比值中等（' + Math.round(ratio * 100) + '%），自身和子调用均有贡献';
    }
    return 'Flat/Cum 比值低（' + Math.round(ratio * 100) + '%），主要是调用入口，热点在下游';
  }

  // 只显示最后两段（包名.函数名），避免过长
  function shortenFunc(name) {
    var parts = name.split('.');
    if (parts.length > 2) return '\u2026' + parts.slice(-2).join('.');
    return name;
  }

  function escHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;')
      .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  // ── 面板渲染 ────────────────────────────────────────────────────────────────

  function createPanel() {
    panel = document.createElement('div');
    panel.id = 'prof-graph-explain';
    panel.setAttribute('data-prof-plugin', 'graph-explain');
    panel.style.cssText = [
      'position:fixed', 'top:60px', 'right:12px',
      'background:#fff', 'border:1px solid #ddd',
      'border-radius:4px', 'padding:12px 14px',
      'min-width:280px', 'max-width:400px',
      'max-height:70vh', 'overflow-y:auto',
      'z-index:9998', 'font-size:13px',
      'font-family:sans-serif', 'line-height:1.5',
      'box-shadow:0 2px 8px rgba(0,0,0,0.12)',
      'display:none',
    ].join(';');
    document.body.appendChild(panel);
  }

  // 从权重字符串中提取数值（支持 kB / MB / B）
  function parseBytes(s) {
    var m = String(s).match(/([\d.]+)\s*(kB|MB|B)?/i);
    if (!m) return 0;
    var v = parseFloat(m[1]);
    var u = (m[2] || '').toLowerCase();
    if (u === 'mb') return v * 1024;
    if (u === 'b')  return v / 1024;
    return v; // kB default
  }

  // 计算某权重占 cum 的百分比，返回字符串如 "26.7%"
  function pctOfCum(weightStr, cumStr) {
    var w = parseBytes(weightStr);
    var c = parseBytes(cumStr);
    if (!c || !w) return '';
    return (w / c * 100).toFixed(1) + '%';
  }

  // 渲染一行树节点，prefix 是 "├── " / "└── " / "│   ├── " 等
  function treeRow(prefix, name, weight, pct, opts) {
    opts = opts || {};
    var nameColor   = opts.nameColor   || '#333';
    var weightColor = opts.weightColor || '#1565c0';
    var html = '<div style="display:flex;align-items:baseline;font-family:monospace;font-size:12px;line-height:1.7">';
    html += '<span style="white-space:pre;color:#bbb;flex-shrink:0">' + escHtml(prefix) + '</span>';
    html += '<span style="flex:1;word-break:break-all;color:' + nameColor + '">' + escHtml(name) + '</span>';
    if (weight) {
      html += '<span style="white-space:nowrap;color:' + weightColor + ';margin-left:6px">' + escHtml(weight) + '</span>';
    }
    if (pct) {
      html += '<span style="white-space:nowrap;color:#aaa;font-size:11px;margin-left:4px">\u2191' + escHtml(pct) + '</span>';
    }
    html += '</div>';
    return html;
  }

  // extra = { graph0, dotId }
  function renderPanel(info, edges, extra) {
    var graph0     = extra && extra.graph0;
    var dotId      = extra && extra.dotId;
    var diag       = diagnoseFlatCum(info.flat, info.cum);
    var hasOuts    = edges.outs.length > 0;
    var hasInlines = edges.inlines && edges.inlines.length > 0;
    var pt         = getProfileType();
    var pkgInfo    = getPackageInfo(info.name);
    var ranking    = graph0 && dotId ? getNodeRanking(graph0, dotId) : null;
    var srcUrl     = buildSourceUrl(info.name);
    var chain      = graph0 && dotId ? buildCallChain(graph0, dotId) : [];
    var html = '', md = '';

    // ── 标题行：函数名 + Source链接 + 复制 + 关闭 ───────────────────────────
    html += '<div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:4px">';
    html += '<strong style="word-break:break-all;font-size:13px;flex:1;line-height:1.4">' + escHtml(info.name) + '</strong>';
    html += '<span style="display:flex;align-items:center;flex-shrink:0;margin-left:6px;gap:3px">';
    if (srcUrl) {
      html += '<a href="' + escHtml(srcUrl) + '" target="_blank" title="Source \u89c6\u56fe" '
            + 'style="text-decoration:none;padding:1px 5px;border:1px solid #ddd;border-radius:3px;'
            + 'font-size:11px;color:#1565c0;background:#fafafa">Source\u2197</a>';
    }
    html += '<span id="prof-explain-copy" title="\u590d\u5236 Markdown" '
          + 'style="cursor:pointer;padding:1px 5px;border:1px solid #ddd;border-radius:3px;'
          + 'font-size:11px;color:#666;background:#fafafa">\uD83D\uDCCB</span>';
    html += '<span id="prof-explain-close" style="cursor:pointer;padding:0 3px;color:#bbb;font-size:14px">\u2715</span>';
    html += '</span></div>';

    // ── 徽标行：包类型 + Profile 类型 + 排名 ────────────────────────────────
    html += '<div style="display:flex;flex-wrap:wrap;gap:4px;align-items:center;margin-bottom:5px">';
    if (pkgInfo) {
      html += '<span style="padding:1px 7px;border-radius:10px;font-size:11px;font-weight:500;'
            + 'color:' + pkgInfo.color + ';background:' + pkgInfo.bg + '">'
            + escHtml(pkgInfo.badge) + '</span>';
    }
    if (pt) {
      html += '<span style="padding:1px 7px;border-radius:10px;font-size:11px;font-weight:500;'
            + 'color:#1a56db;background:#e8f0fe">' + escHtml(pt.label) + '</span>';
      if (pt.desc) {
        html += '<span style="color:#aaa;font-size:11px">' + escHtml(pt.desc) + '</span>';
      }
    }
    if (ranking) {
      html += '<span style="color:#888;font-size:11px;margin-left:auto">cum \u7b2c '
            + ranking.rank + ' / ' + ranking.total + '</span>';
    }
    html += '</div>';

    // ── md：与面板所见保持一致，无主观框架 ────────────────────────────────────
    md += '\u51fd\u6570\uff1a' + info.name + '\n';
    if (pkgInfo) md += '\u5305\u7c7b\u578b\uff1a' + pkgInfo.badge + '\n';
    if (pt)      md += 'Profile\uff1a' + pt.label + (pt.desc ? '\uff08' + pt.desc + '\uff09' : '') + '\n';
    if (ranking) md += 'cum \u6392\u540d\uff1a\u7b2c ' + ranking.rank + ' / ' + ranking.total + '\n';

    // ── Flat / Cum ──────────────────────────────────────────────────────────
    html += '<hr style="margin:4px 0 6px;border:none;border-top:1px solid #eee">';
    html += '<div style="display:flex;gap:16px;font-size:12px">';
    html += '<div><span style="color:#888">\u81ea\u8eab </span><strong>' + escHtml(info.flat || '0') + '</strong>'
          + '<span style="color:#aaa;font-size:10px;margin-left:2px">\u76f4\u63a5\u5206\u914d</span></div>';
    html += '<div><span style="color:#888">\u7d2f\u8ba1 </span><strong>' + escHtml(info.cum || '\u2014') + '</strong>'
          + '<span style="color:#aaa;font-size:10px;margin-left:2px">\u542b\u5b50\u8c03\u7528</span></div>';
    html += '</div>';
    md += '\n\u81ea\u8eab\uff08flat\uff09\uff1a' + (info.flat || '0') + '\u3000\u76f4\u63a5\u5206\u914d\n';
    md += '\u7d2f\u8ba1\uff08cum\uff09\uff1a'   + (info.cum  || '\u2014') + '\u3000\u542b\u5168\u90e8\u5b50\u8c03\u7528\n';

    // ── 完整调用路径 ────────────────────────────────────────────────────────
    if (chain.length > 1) {
      html += '<div style="margin-top:9px;font-size:12px;font-weight:600;color:#555">'
            + '\uD83D\uDDFA\uFE0F \u8c03\u7528\u8def\u5f84</div>';
      html += '<div style="font-family:monospace;font-size:11px;color:#666;line-height:1.8;'
            + 'background:#fafafa;border-radius:3px;padding:4px 6px;margin-top:2px;word-break:break-all">';
      html += chain.map(function (n, i) {
        var isLast = i === chain.length - 1;
        var name   = shortenFunc(n.func);
        return isLast
          ? '<span style="color:#c62828;font-weight:600">' + escHtml(name) + '</span>'
          : escHtml(name);
      }).join(' \u2192 ');
      html += '</div>';
      md += '\n\uD83D\uDDFA\uFE0F \u8c03\u7528\u8def\u5f84\n';
      md += chain.map(function (n) { return n.func; }).join(' \u2192 ') + '\n';
    }

    // ── 调用者 tree ─────────────────────────────────────────────────────────
    if (edges.ins.length > 0) {
      html += '<div style="margin-top:9px;font-size:12px;font-weight:600;color:#555">\uD83D\uDCE5 \u8c03\u7528\u8005</div>';
      md  += '\n\uD83D\uDCE5 \u8c03\u7528\u8005\n';
      edges.ins.forEach(function (e, i) {
        var isLast = i === edges.ins.length - 1;
        var pre    = isLast ? '\u2514\u2500\u2500 ' : '\u251C\u2500\u2500 ';
        html += treeRow(pre, e.func, e.weight, '', { weightColor: '#c62828' });
        md   += pre + e.func + '  ' + e.weight + '\n';
      });
    }

    // ── 被调用 tree ─────────────────────────────────────────────────────────
    if (hasOuts || hasInlines) {
      html += '<div style="margin-top:9px;font-size:12px;font-weight:600;color:#555">'
            + '\uD83D\uDCE4 \u88ab\u8c03\u7528'
            + '<span style="font-weight:400;color:#bbb;margin-left:4px">\u6574\u4f53 '
            + escHtml(info.cum || '') + '</span></div>';
      md  += '\n\uD83D\uDCE4 \u88ab\u8c03\u7528\uff08\u6574\u4f53 ' + (info.cum || '') + '\uff09\n';

      var topItems   = edges.outs.slice();
      if (hasInlines) topItems.push({ isInlineGroup: true });

      topItems.forEach(function (item, i) {
        var isLast  = i === topItems.length - 1;
        var pre     = isLast ? '\u2514\u2500\u2500 ' : '\u251C\u2500\u2500 ';
        var contPre = isLast ? '    ' : '\u2502   ';

        if (item.isInlineGroup) {
          var flatPct = pctOfCum(info.flat, info.cum);
          html += treeRow(pre, '\u81ea\u8eab\u76f4\u63a5\u5206\u914d\uff08flat\uff09', info.flat, flatPct, { nameColor: '#666' });
          md   += pre + '\u81ea\u8eab\u76f4\u63a5\u5206\u914d\uff08flat\uff09  ' + (info.flat || '') + (flatPct ? '  \u2191' + flatPct : '') + '\n';
          edges.inlines.forEach(function (e, j) {
            var subLast = j === edges.inlines.length - 1;
            var subPre  = contPre + (subLast ? '\u2514\u2500\u2500 ' : '\u251C\u2500\u2500 ');
            var pct     = pctOfCum(e.weight, info.cum);
            var label   = e.label.replace(/\.\./g, '~') + ' \u5c3a\u5bf8';
            html += treeRow(subPre, label, e.weight, pct, { nameColor: '#888' });
            md   += subPre + label + '  ' + e.weight + (pct ? '  \u2191' + pct : '') + '\n';
          });
        } else {
          var pct = pctOfCum(item.weight, info.cum);
          html += treeRow(pre, item.func, item.weight, pct);
          md   += pre + item.func + '  ' + item.weight + (pct ? '  \u2191' + pct : '') + '\n';
        }
      });
    }

    // ── 诊断 ────────────────────────────────────────────────────────────────
    if (diag) {
      html += '<hr style="margin:8px 0 5px;border:none;border-top:1px solid #eee">';
      html += '<div style="color:#666;font-size:12px;line-height:1.5">\uD83D\uDCA1 ' + escHtml(diag) + '</div>';
      md   += '\n\uD83D\uDCA1 ' + diag + '\n';
    }

    panel.innerHTML = html;
    panel._copyText = md;

    var closeBtn = panel.querySelector('#prof-explain-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', function () { panel.style.display = 'none'; });
    }
    var copyBtn = panel.querySelector('#prof-explain-copy');
    if (copyBtn) {
      copyBtn.addEventListener('click', function () {
        var t = panel._copyText || '';
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(t).then(function () {
            copyBtn.textContent = '\u2713';
            setTimeout(function () { copyBtn.textContent = '\uD83D\uDCCB'; }, 1500);
          }).catch(function () { fallbackCopy(t, copyBtn); });
        } else {
          fallbackCopy(t, copyBtn);
        }
      });
    }
  }

  function fallbackCopy(text, btn) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
      if (btn) { btn.textContent = '\u2713'; setTimeout(function () { btn.textContent = '\uD83D\uDCCB'; }, 1500); }
    } catch (e) { /* ignore */ }
    document.body.removeChild(ta);
  }

  function showPanel(info, edges, extra) {
    if (!panel) createPanel();
    renderPanel(info, edges, extra);
    panel.style.display = 'block';
  }

  // ── 事件绑定 ────────────────────────────────────────────────────────────────

  var _bound = false;

  function init() {
    var graph0 = document.getElementById('graph0');
    if (!graph0) return;
    // graph0 是 <g>，其父元素是 <svg>
    var svg = graph0.parentElement;
    if (!svg) return;
    // 避免重复绑定
    if (svg._graphExplainBound) return;
    svg._graphExplainBound = true;

    svg.addEventListener('click', function (e) {
      // 向上找到 graph0 的直接子元素（节点 <g>）
      var el = e.target;
      while (el && el.parentElement !== graph0) {
        el = el.parentElement;
      }
      if (!el || !el.id || el.id.indexOf('node') !== 0) {
        // 以下情况保持面板可见，不关闭：
        //   1. edge 元素（边路径）
        //   2. nodelet 节点（id = NN3_0 格式，是 inline 函数小方块）
        //   3. 边 label 元素（id 含 "edge"，如 a_edge41-label）
        var isEdge     = el && el.classList && el.classList.contains('edge');
        var isNodelet  = el && el.id && /^NN\d+/.test(el.id);
        var isEdgeElem = el && el.id && el.id.indexOf('edge') !== -1;
        if (!isEdge && !isNodelet && !isEdgeElem && panel) {
          panel.style.display = 'none';
        }
        return;
      }

      var titleInfo = parseNodeTitle(el);
      var textInfo  = parseNodeTexts(el);
      var funcName  = titleInfo.name || '';
      if (!funcName) return;

      var info = {
        name: funcName,
        flat: textInfo.flat,
        cum:  textInfo.cum || ('(' + titleInfo.cum + ')'),
      };

      var dotId = toDotId(el.id);
      var edges = findEdges(graph0, dotId);
      showPanel(info, edges, { graph0: graph0, dotId: dotId });

      // 阻止事件冒泡，避免 pprof 原生点击逻辑关闭面板
      e.stopPropagation();
    });
  }

  // ── 注册 ─────────────────────────────────────────────────────────────────────

  P.register({
    name: 'graph-explain',
    version: '1.0.0',
    views: ['graph'],
    init: function () { init(); },
    onViewChange: function (view) {
      if (view === 'graph') {
        init();
      } else if (panel) {
        panel.style.display = 'none';
      }
    },
  });

})(window.ProfPlugin);
