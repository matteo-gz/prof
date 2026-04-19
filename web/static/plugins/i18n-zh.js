(function (P) {
  'use strict';

  var dict = {
    // ===== 导航菜单 =====
    'View':           '视图',
    'Top':            '排名',
    'Graph':          '调用图',
    'Flame Graph':    '火焰图',
    'Peek':           '窥视',
    'Source':         '源码',
    'Disassemble':    '反汇编',

    'Sample':         '采样类型',

    // ===== Refine / Config 菜单 =====
    'Refine':         '筛选',
    'Focus':          '聚焦',
    'Ignore':         '忽略',
    'Hide':           '隐藏',
    'Show':           '显示',
    'Show from':      '从此显示',
    'Reset':          '重置',

    'Config':         '配置',
    'Save as ...':    '另存为…',
    'Download':       '下载',

    // ===== Top 表头 =====
    'Flat':           '自身',
    'Flat%':          '自身%',
    'Sum%':           '累计%',
    'Cum':            '累计',
    'Cum%':           '累计占比%',
    'Name':           '函数名',
    'Inlined?':       '内联?',

    // ===== 对话框 =====
    'Save options as':  '保存配置为',
    'Cancel':           '取消',
    'Save':             '保存',
    'Delete config':    '删除配置',
    'Delete':           '删除',

    // ===== Flame Graph 结构文案 =====
    'Search regexp':              '搜索正则',
    'Reset zoom':                 '重置缩放',

    // ===== Flame Graph action menu =====
    'Show source code':           '查看源码',
    'Show source in new tab':     '在新标签页查看源码',
  };

  // Graph SVG 字幕：含动态数字，用正则 partial-match
  var partialPatterns = [
    {
      re: /^Showing nodes accounting for (.+?), (.+?) of (.+?) total(.*)$/,
      fn: function (m) {
        return '显示节点占 ' + m[1] + '，共 ' + m[3] + ' 的 ' + m[2] + m[4];
      }
    },
    {
      re: /^Dropped (\d+) nodes? \(cum <= (.+?)\)(.*)$/,
      fn: function (m) {
        return '已省略 ' + m[1] + ' 个节点（cum ≤ ' + m[2] + '）' + m[3];
      }
    },
    {
      re: /^Dropped (\d+) edges? \(freq <= (.+?)\)(.*)$/,
      fn: function (m) {
        return '已省略 ' + m[1] + ' 条边（freq ≤ ' + m[2] + '）' + m[3];
      }
    },
  ];

  function translatePartial(text) {
    for (var i = 0; i < partialPatterns.length; i++) {
      var m = text.match(partialPatterns[i].re);
      if (m) return partialPatterns[i].fn(m);
    }
    return null;
  }

  function translateTextNode(node) {
    if (node.nodeType === 3) {
      var text = node.textContent.trim();
      if (text && dict[text]) {
        node.textContent = node.textContent.replace(text, dict[text]);
      }
      return;
    }
    if (node.nodeType !== 1) return;
    var tag = node.tagName;
    if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'CODE' || tag === 'PRE') return;
    // 火焰图函数块在 #stack-holder 内，函数名不翻译
    if (node.id === 'stack-holder' || node.id === 'stack-chart') return;

    if (node.children.length === 0) {
      var t = node.textContent.trim();
      if (t && dict[t]) {
        node.textContent = dict[t];
        return;
      }
      var partial = translatePartial(t);
      if (partial) {
        node.textContent = partial;
      }
      return;
    }
    for (var i = 0; i < node.childNodes.length; i++) {
      translateTextNode(node.childNodes[i]);
    }
  }

  function translateAttrs(root) {
    root.querySelectorAll('[title]').forEach(function (el) {
      var t = el.getAttribute('title').trim();
      if (dict[t]) el.setAttribute('title', dict[t]);
    });
    root.querySelectorAll('[placeholder]').forEach(function (el) {
      var p = el.getAttribute('placeholder').trim();
      if (dict[p]) el.setAttribute('placeholder', dict[p]);
    });
  }

  function translatePage() {
    translateTextNode(document.body);
    translateAttrs(document);
  }

  P.register({
    name: 'i18n-zh',
    version: '1.3.0',
    views: ['*'],

    init: function () {
      translatePage();
    },

    onViewChange: function () {
      translatePage();
    },

    onMutation: function (nodes) {
      for (var i = 0; i < nodes.length; i++) {
        // 火焰图函数块在 #stack-holder 内，跳过
        var n = nodes[i];
        if (n.closest && n.closest('#stack-holder')) continue;
        translateTextNode(n);
      }
    }
  });

})(window.ProfPlugin);
