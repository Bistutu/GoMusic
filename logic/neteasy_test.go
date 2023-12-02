package logic

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"GoMusic/common/utils"
)

const (
	V1 = "http://163cn.tv/zoIxm3"
	V2 = "https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998"
	V3 = "https://music.163.com/playlist?id=477577176&userid=341246998"
	V4 = "分享Mad_Cat_的歌单《Mad_Cat_喜欢的音乐》http://163cn.tv/aSl9Z1 (@网易云音乐)"
)

func TestRegex(t *testing.T) {
	sample := []string{V1, V2, V3, V4}
	urlPattern := `http[s]?://[^ ]+`
	re := regexp.MustCompile(urlPattern)
	for _, v := range sample {
		fmt.Println("Extracted URL:", re.FindString(v))
	}
}

func TestBracketRegex(t *testing.T) {
	fmt.Println(utils.StandardSongName("理想三旬（女声版） - 藤柒吖"))
	fmt.Println(utils.StandardSongName("小酒窝(Live) - 蔡卓妍 / 林俊杰"))
	fmt.Println(utils.StandardSongName("最后一页（完整版） - 洛尘鞅_"))
	fmt.Println(utils.StandardSongName("知我（抒情版） - 尘ah."))
	fmt.Println(utils.StandardSongName("幻听（女声版） - 星月酱"))
}

func TestDiscover(t *testing.T) {
	sample := []string{V1, V2, V3, V4}
	for _, v := range sample {
		discover, err := NetEasyDiscover(v)
		assert.NoError(t, err)
		t.Log(discover)
	}
}
