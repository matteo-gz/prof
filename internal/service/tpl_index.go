package service

const tpl_index = `<html>
<head>
    <title>prof</title>
    <link href="./bootstrap.min.css" rel="stylesheet"
          crossorigin="anonymous">
</head>
<body>
<div class="container">
    <ul class="nav">
        <li class="nav-item">
            <a class="nav-link disabled" href="javascript:">Home</a>
        </li>
        <li class="nav-item">
            <a class="nav-link" href="/history">file list</a>
        </li>
        <li class="nav-item">
            <a class="nav-link" href="/person/curl">person</a>
        </li>
    </ul>

    <div class="row">
        <div class="col">
            <!--    curl one-->
            <div class="mb-3">
                <legend>curl one</legend>
                <div class="mb-3">
                    <label class="form-label">url</label>
                    <input class="form-control" name="url_one" type="text" placeholder="url">
                    <div class="form-text">demo: http://127.0.0.1:8202/internal/debug/pprof/goroutine</div>
                </div>
                <button type="submit" class="btn btn-primary" id="fetchdata1">submit</button>
                <div id="result3"></div>
            </div>
        </div>
        <div class="col">
            <!--    upload -->
            <div class="mb-3">
                <legend>upload file</legend>
                <div class="mb-3">
                    <label class="form-label">file</label>
                    <input class="form-control" type="file" name="file" accept="*">
                    <div class="form-text"></div>
                </div>
                <button type="submit" class="btn btn-primary" id="subf">submit</button>
                <div id="result2"></div>
            </div>
        </div>


    </div>
    <div class="row">
        <div class="col">
            <!--curl batch-->
            <div class="mb-3">
                <legend>curl batch</legend>
                <div class="mb-3">
                    <label class="form-label">url</label>
                    <input class="form-control" type="text" name="url" placeholder="url"/>
                    <div class="form-text">demo: http://127.0.0.1:8202/internal/debug/pprof?seconds=2</div>
                </div>
            </div>
            <button type="submit" class="btn btn-primary" id="fetchdata">submit</button>
            <div class="p-3 mb-2 bg-light text-dark">
                <span>http://127.0.0.1:8202/internal/debug/pprof/allocs?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/block?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/cmdline?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/goroutine?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/heap?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/mutex?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/threadcreate?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/profile?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/trace?seconds=2</span>
                <span>http://127.0.0.1:8202/internal/debug/pprof/symbol?seconds=2</span>
            </div>
        </div>
        <div class="col">
            <div id="result"></div>
        </div>
    </div>

</div>
</body>


<script>
    let runlock = false;
    let runlock2 = false;
    let uploadlock = false;
    let key = "id"
    const run = function () {
        tpl_res1.innerText = "load..."
        let u = document.querySelector('input[name="url"]').value
        let uaf = encodeURIComponent(u)
        if (u === "") {
            tpl_res1.innerText = "url not"
            return
        }
        let pass = 0
        let itv = setInterval(function () {
            ++pass
            tpl_res1.innerText = "fetching..." + pass
        }, 1000);
        if (runlock) {
            return
        }
        runlock = true
        let formdata = new FormData();
        formdata.append("url", uaf);
        let requestOptions = {
            method: 'POST',
            body: formdata,
        }
        fetch("/opt/run", requestOptions)
            .then((res) => {
                window.clearInterval(itv)
                runlock = false
                return res.json()
            })
            .then((data) => {
                let t = "<ol>"
                for (let i of data.res) {
                    t += "<li>" + i + "</li>"
                }
                t += "</ol>"
                tpl_res1.innerHTML = t
                store(data.id, t, u, "curl batch")
                document.querySelector('input[name="url"]').value = ""
            })
            .catch((error) => {
                console.log(error)
            });
    };
    const run1 = function () {
        tpl_res3.innerText = "load..."
        let u = document.querySelector('input[name="url_one"]').value
        let uaf = encodeURIComponent(u)
        if (u === "") {
            tpl_res3.innerText = "url not"
            return
        }
        let pass = 0
        let itv = setInterval(function () {
            ++pass
            tpl_res3.innerText = "fetching..." + pass
        }, 1000);
        if (runlock2) {
            return
        }
        runlock2 = true
        let formdata = new FormData();
        formdata.append("url", uaf);
        let requestOptions = {
            method: 'POST',
            body: formdata,
        }
        fetch("/opt/run1", requestOptions)
            .then((res) => {
                window.clearInterval(itv)
                runlock2 = false
                return res.json()
            })
            .then((data) => {
                let t = "<ol>"
                for (let i of data.res) {
                    t += "<li>" + i + "</li>"
                }
                t += "</ol>"
                tpl_res3.innerHTML = t
                store(data.id, t, u, "curl one")
                document.querySelector('input[name="url_one"]').value = ""
            })
            .catch((error) => {
                console.log(error)
            });
    };
    const store = function (index, data, name, act) {
        let res = localStorage.getItem(key)
        let a = []
        if (res !== null) {
            a = JSON.parse(res)
        }
        a.push(index)
        localStorage.setItem(key, JSON.stringify(a))
        let obj = {
            index: index,
            data: data,
            name: name,
            act: act,
            now: getNowTime(),
        }
        localStorage.setItem(index, JSON.stringify(obj));
    }

    const upload = function () {
        let formdata = new FormData();
        let ff = document.querySelector('input[name="file"]').files[0]
        if (ff == null) {
            tpl_res2.innerText = "file empty..."
            return;
        }
        tpl_res2.innerText = "load..."
        formdata.append("file", ff, ff.name);
        let requestOptions = {
            method: 'POST',
            body: formdata,
            redirect: 'follow'
        };
        tpl_res2.innerText = "fetching..."

        if (uploadlock) {
            return
        }
        uploadlock = true

        fetch("/opt/upload", requestOptions)
            .then(async (res) => {
                uploadlock = false
                if (res.status === 200) {
                    let data = await res.json()
                    return data
                } else {
                    let data = await res.text()
                    console.log(data)
                    throw new Error(data)
                }
            })
            .then((data) => {
                if (data === undefined) {
                    return
                }
                let t = "<ol>"
                for (let i of data.res) {
                    t += "<li>" + i + "</li>"
                }
                t += "</ol>"
                tpl_res2.innerHTML = t
                store(data.id, t, ff.name, "upload")

                document.querySelector('input[name="file"]').value = ""
            }).catch(error => {
            tpl_res2.innerHTML = error.toString()
        });
    }

    function getNowTime() {
        let date = new Date();
        let year = date.getFullYear();
        let month = date.getMonth() + 1;
        let day = date.getDate();
        let hour = date.getHours();
        let minute = date.getMinutes();
        let second = date.getSeconds();
        return 'time：' + year + '-' + addZero(month) + '-' + addZero(day) + ' ' + addZero(hour) + ':' + addZero(minute) + ':' + addZero(second);
    }

    function addZero(s) {
        return s < 10 ? ('0' + s) : s;
    }

    document.querySelector('#fetchdata').addEventListener('click', run)
    document.querySelector('#fetchdata1').addEventListener('click', run1)
    document.querySelector('#subf').addEventListener('click', upload)
    const tpl_res1 = document.querySelector("#result")
    const tpl_res2 = document.querySelector("#result2")
    const tpl_res3 = document.querySelector("#result3")
</script>
</html>`
