package utils

import (
	"fmt"
	"testing"
)

func TestEncry(t *testing.T) {
	info := `{"req_0":{"module":"music.srfDissInfo.aiDissInfo","method":"uniform_get_Dissinfo","param":{"disstid":7364061065,"enc_host_uin":"","tag":1,"userinfo":1,"song_begin":1,"song_num":1024}},"comm":{"g_tk":5381,"uin":0,"format":"json","platform":"h5"}}`
	signNative, _ := GetSign(info)
	fmt.Println(signNative)
	sign := Encrypt(info)
	fmt.Println(sign)
}

func BenchmarkName(b *testing.B) {
	msg := `{"req_0":{"module":"music.srfDissInfo.aiDissInfo","method":"uniform_get_Dissinfo","param":{"disstid":7364061065,"enc_host_uin":"","tag":1,"userinfo":1,"song_begin":1,"song_num":1024}},"comm":{"g_tk":5381,"uin":0,"format":"json","platform":"h5"}}`
	b.Run("js", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetSign(msg)
		}
	})
	b.Run("go", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Encrypt(msg)
		}
	})

}
