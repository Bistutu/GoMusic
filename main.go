package main

import (
	"fmt"

	"GoMusic/common/config"
	"GoMusic/initialize"
	"GoMusic/initialize/log"
)

func main() {
	r := initialize.NewRouter()
	if err := r.Run(fmt.Sprintf(":%d", config.AllConfig.Port)); err != nil {
		log.Errorf("fail to run server: %v", err)
		panic(err)
	}
}
