package service

const tpl_person_curl = `<html>
<head>
    <title>prof</title>
    <link href="../bootstrap.min.css" rel="stylesheet"
          crossorigin="anonymous">
</head>
<body>
<div class="container">
    <ul class="nav">
        <li class="nav-item">
            <a class="nav-link" href="/">Home</a>
        </li>
        <li class="nav-item">
            <a class="nav-link disabled">person</a>
        </li>
    </ul>
    <div class="row">
        <div class="col">
            <button class="btn btn-primary" type="button" id="clear">clear all</button>
        </div>
    </div>
    <div class="row">
        <div class="col">
            <div id="result3"></div>
        </div>
    </div>
</div>
</body>
<script>
    let key = "id"
    const tpl_res3 = document.querySelector("#result3")
    const deleteStore = function (index) {
        let msg = "wanna delete?";
        if (confirm(msg) === true) {
            localStorage.removeItem(index)
            loadStore()
            return true;
        } else {
            return false;
        }
    }
    const loadStore = function () {
        let res = localStorage.getItem(key)
        let a = []
        if (res !== null) {
            a = JSON.parse(res)
        }
        a = a.reverse()
        let t = ""
        t += "<div>"
        t += "<ol>"
        for (let i of a) {
            let obj = JSON.parse(localStorage.getItem(i))
            if (obj === null) {
                continue
            }
            let tpl = "<span>"
            let idx = obj.index
            tpl += "<button type=\"button\" onclick='deleteStore(" + idx + ")'>delete->" + idx + "</button>"
            tpl += "<div>" + obj.now + "</div>"
            tpl += "<div>act: " + obj.act + "</div>"
            tpl += "<div>name: " + obj.name + "</div>"
            tpl += obj.data
            tpl += "</span>"
            t += "<li>" + tpl + "</li>"
        }
        t += "</ol>"
        t += "</div>"
        tpl_res3.innerHTML = t
    }
    loadStore()
    const clearAll = function () {
        let msg = "wanna delete?";
        if (confirm(msg) === true) {
            localStorage.clear()
            loadStore()
            return true;
        } else {
            return false;
        }
    }
    document.querySelector('#clear').addEventListener('click', clearAll)
</script>
</html>`
