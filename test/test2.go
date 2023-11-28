package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

func main() {
	// 替换以下路径为您的文件路径
	filePath := "/Users/miankang.chen/Desktop/zlist.html"

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return
	}
	defer file.Close()

	// 正则表达式匹配6位连续数字
	re := regexp.MustCompile(`\d{6}`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			fmt.Println(line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("读取文件时出错:", err)
	}
}
