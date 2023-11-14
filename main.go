package main

import (
	"fmt"

	_ "GoMusic/repo/db"

	"GoMusic/initialize"
	"GoMusic/initialize/log"
)

func main() {
	r := initialize.NewRouter()
	if err := r.Run(fmt.Sprintf(":%d", 8081)); err != nil {
		log.Errorf("fail to run server: %v", err)
		panic(err)
	}
}
