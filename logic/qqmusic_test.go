package logic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQQMusicDiscover(t *testing.T) {
	// https://c6.y.qq.com/base/fcgi-bin/u?__=4V33zWKDE3tI
	// https://y.qq.com/n/ryqq/playlist/7364061065
	// https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=oKE57evqoiEPoz**&id=1596010000&appversion=120801&ADTAG=wxfshare&appshare=iphone_wx
	// https://i.y.qq.com/n2/m/share/details/taoge.html?platform=11&appshare=android_qq&appversion=12090008&hosteuin=oK6kowEAoK4z7eSk7eEloKCFoz**&id=5204875759&ADTAG=wxfshare
	// https://i.y.qq.com/n2/m/share/details/taoge.html?id=9094523921	// 12.11格式
	// https://i.y.qq.com/n2/m/share/details/taoge.html?id=8177163754

	t.Run("v1", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://c6.y.qq.com/base/fcgi-bin/u?__=4V33zWKDE3tI")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v2", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://y.qq.com/n/ryqq/playlist/7364061065")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v3", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=oKE57evqoiEPoz**&id=1596010000&appversion=120801&ADTAG=wxfshare&appshare=iphone_wx")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v4", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?platform=11&appshare=android_qq&appversion=12090008&hosteuin=oK6kowEAoK4z7eSk7eEloKCFoz**&id=5204875759&ADTAG=wxfshare")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v5", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?id=9094523921")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v6", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?id=8177163754")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
}
