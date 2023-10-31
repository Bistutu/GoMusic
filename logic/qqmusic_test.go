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
}
