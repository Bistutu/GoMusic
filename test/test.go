package main

import (
	"fmt"
	"regexp"
)

func main() {
	bracketsPattern := `\s\(.*\)|\s【.*】`

	bracketsRegex := regexp.MustCompile(bracketsPattern)
	name := "最甜情歌 (女生版) - 一玟"

	name = bracketsRegex.ReplaceAllString(name, "")

	fmt.Println(name) // 输出：最甜情歌 女生版 - 一玟
}
