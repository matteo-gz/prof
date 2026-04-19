(function () {
  'use strict';

  var DEFAULT_LANG = 'zh';
  var LANG_KEY = 'prof-lang';

  var lang = localStorage.getItem(LANG_KEY) || DEFAULT_LANG;
  var translations = {};

  function loadTranslations(l) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/static/i18n/' + l + '.json', false); // async=false intentional: local tool
    try {
      xhr.send();
      if (xhr.status === 200) return JSON.parse(xhr.responseText);
    } catch (e) {}
    return {};
  }

  translations = loadTranslations(lang);

  window.t = function (key) {
    return translations[key] !== undefined ? translations[key] : key;
  };

  window.ProfI18n = {
    lang: lang,
    t: window.t,
    setLang: function (newLang) {
      localStorage.setItem(LANG_KEY, newLang);
      location.reload();
    },
    getLang: function () { return lang; },
  };

  function applyTranslations() {
    document.querySelectorAll('[data-i18n]').forEach(function (el) {
      var key = el.getAttribute('data-i18n');
      if (translations[key] !== undefined) el.textContent = translations[key];
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(function (el) {
      var key = el.getAttribute('data-i18n-placeholder');
      if (translations[key] !== undefined) el.placeholder = translations[key];
    });
  }

  function injectSwitcher() {
    var nav = document.querySelector('.navbar-nav');
    if (!nav) return;
    if (document.getElementById('prof-lang-switcher')) return;

    var li = document.createElement('li');
    li.className = 'nav-item';
    li.id = 'prof-lang-switcher';
    li.style.cssText = 'cursor:pointer;';

    var a = document.createElement('a');
    a.className = 'nav-link';
    a.style.cssText = 'user-select:none;';
    a.textContent = lang === 'zh' ? 'EN' : '中';
    a.title = lang === 'zh' ? 'Switch to English' : '切换为中文';
    a.addEventListener('click', function () {
      ProfI18n.setLang(lang === 'zh' ? 'en' : 'zh');
    });

    li.appendChild(a);
    nav.appendChild(li);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      applyTranslations();
      injectSwitcher();
    });
  } else {
    applyTranslations();
    injectSwitcher();
  }
})();
