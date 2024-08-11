package service

const tpl_list = `<html>
<head>
    <link href="./bootstrap.min.css" rel="stylesheet"
          crossorigin="anonymous">
</head>
<body>
<div class="container">
    <ul class="nav">
        <li class="nav-item">
            <a class="nav-link" href="/">Home</a>
        </li>
        <li class="nav-item">
            <a class="nav-link disabled" href="/history">file list</a>
        </li>

    </ul>
    <div class="row">
        <a href="#" onclick="window.history.go(-1)">{{.dir}}</a>
    </div>
    <div class="row">
        <ol class="list-group list-group-flush">
            {{$dir := .dir}}
            {{range $_ := .list}}
            <li><a href="/history?dir={{$dir}}/{{.}}">{{.}}</a></li>
            {{end}}
        </ol>

    </div>
    <div class="row">
        <ol class="list-group list-group-flush">
            {{range $_ := .files}}
            <li><a target="_blank" href="/file?dir={{$dir}}/{{.}}">{{.}}</a></li>
            {{end}}
        </ol>
    </div>
</div>

</body>

</html>`
