package utils

import (
	_ "embed"

	"github.com/robertkrimen/otto"

	"GoMusic/misc/log"
)

//go:embed qqmusic_encrypt.js
var qqmusicJS string

var vm = otto.New()

func init() {
	if _, err := vm.Run(qqmusicJS); err != nil {
		log.Errorf("fail to run js: %v", err)
		panic(err)
	}
}

func GetSign(data string) (string, error) {
	value, err := vm.Call("get_sign", nil, data)
	if err != nil {
		log.Errorf("fail to call js: %v", err)
		return "", err
	}
	return value.String(), nil
}
