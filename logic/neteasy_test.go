package logic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"GoMusic/common/utils"
)

func TestRegex(t *testing.T) {
	// http://163cn.tv/zoIxm3
	// https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998
	// https://music.163.com/playlist?id=477577176&userid=341246998
}

func TestBracketRegex(t *testing.T) {
	fmt.Println(utils.StandardSongName("理想三旬（女声版） - 藤柒吖"))
	fmt.Println(utils.StandardSongName("小酒窝(Live) - 蔡卓妍 / 林俊杰"))
	fmt.Println(utils.StandardSongName("最后一页（完整版） - 洛尘鞅_"))
	fmt.Println(utils.StandardSongName("知我（抒情版） - 尘ah."))
	fmt.Println(utils.StandardSongName("幻听（女声版） - 星月酱"))
}

func TestDiscover(t *testing.T) {
	discover, err := NetEasyDiscover("http://163cn.tv/zoIxm3")
	assert.NoError(t, err)
	t.Log(discover)
}
