package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	msg := `{"Name":"zs","Age":18,"Msg":"haha"}`
	p := &P{}
	err := json.Unmarshal([]byte(msg), p)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(p)
}

type P struct {
	Name string
	Age  int
}
