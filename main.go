package main

import (
	"fmt"

	"GoMusic/handler"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	_ "GoMusic/repo/db"
)

func main() {
	r := handler.NewRouter()
	if err := r.Run(fmt.Sprintf(models.PortFormat, models.Port)); err != nil {
		log.Errorf("fail to run server: %v", err)
		panic(err)
	}
}
