package utils

import (
	_ "embed"
	"log"

	"github.com/robertkrimen/otto"
)

//go:embed qqmusic_encrypt.js
var qqmusicJS string

var vm = otto.New()

func init() {
	if _, err := vm.Run(qqmusicJS); err != nil {
		log.Fatalf("fail to run js: %v", err)
	}
}

func GetSign(data string) (string, error) {
	value, err := vm.Call("get_sign", nil, data)
	if err != nil {
		log.Printf("fail to call js: %v", err)
		return "", err
	}
	return value.String(), nil
}
