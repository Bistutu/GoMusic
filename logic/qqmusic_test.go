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
	// https://y.qq.com/n/ryqq/playlist/1953563505
	// https://i.y.qq.com/n2/m/share/details/taoge.html?id=9115464345&hosteuin=
	// https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=owEkNKSzNK65&id=930054744&appversion=130000&ADTAG=qfshare&source=qq&appshare=iphone
	// https://c6.y.qq.com/base/fcgi-bin/u?__=XogXh1TLpP1t
	// https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=ownioivi7wvq7n**&id=8683730831&appversion=120801&ADTAG=wxfshare&appshare=iphone_wx

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
	t.Run("v7", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://y.qq.com/n/ryqq/playlist/1953563505")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v8", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?id=9115464345&hosteuin=")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v9", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=owEkNKSzNK65&id=930054744&appversion=130000&ADTAG=qfshare&source=qq&appshare=iphone")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("v10", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://c6.y.qq.com/base/fcgi-bin/u?__=XogXh1TLpP1t")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
	t.Run("V11", func(t *testing.T) {
		discover, err := QQMusicDiscover("https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=ownioivi7wvq7n**&id=8683730831&appversion=120801&ADTAG=wxfshare&appshare=iphone_wx")
		assert.NoError(t, err)
		fmt.Println(discover)
	})
}
