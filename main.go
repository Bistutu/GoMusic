package main

import (
	"fmt"

	"GoMusic/initialize"
	"GoMusic/initialize/log"
)

func main() {
	r := initialize.NewRouter()
	if err := r.Run(fmt.Sprintf(":%d", 80)); err != nil {
		log.Errorf("fail to run server: %v", err)
		panic(err)
	}
}
