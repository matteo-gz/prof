package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	f("index")
	f("person_curl")
	f("list")
}

func f(name string) {
	index, _ := ioutil.ReadFile(fmt.Sprintf("./web/template/%s.html", name))
	index_tpl := `package service

const tpl_%s=%s%s%s`
	s := fmt.Sprintf(index_tpl, name, "`", string(index), "`")
	_ = ioutil.WriteFile(fmt.Sprintf("./internal/service/tpl_%s.go", name), []byte(s), os.ModeAppend)
}
