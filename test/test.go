package main

import (
	"fmt"
	"regexp"
)

func main() {
	netEasyRegex := "http://163cn.tv/\\w{6}"
	compile, _ := regexp.Compile(netEasyRegex)
	fmt.Println(compile.MatchString("http://163cn.tv/zoIxm3"))
}
