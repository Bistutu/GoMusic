package logic

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"GoMusic/misc/utils"
)

const (
	V1 = "http://163cn.tv/zoIxm3"
	V5 = "http://music.163.com/playlist/2275447155/434174568/?userid=440609461"

	V2 = "https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998"
	V3 = "https://music.163.com/playlist?id=477577176&userid=341246998"
	V4 = "分享Mad_Cat_的歌单《Mad_Cat_喜欢的音乐》http://163cn.tv/aSl9Z1 (@网易云音乐)"
)

func TestRegex(t *testing.T) {
	t.Run("Extracted URL", func(t *testing.T) {
		sample := []string{V1, V2, V3, V4, V5}
		urlPattern := `http[s]?://[^ ]+`
		re := regexp.MustCompile(urlPattern)
		for _, v := range sample {
			fmt.Println("Extracted URL:", re.FindString(v))
		}
	})
	t.Run("Extracted ID", func(t *testing.T) {
		sample := []string{V1, V2, V3, V4, V5}
		re := regexp.MustCompile(`playlist/(\d+)`)
		// 在字符串中查找第一个匹配项
		for _, v := range sample {
			match := re.FindStringSubmatch(v)
			// 检查是否找到匹配项，并打印
			if len(match) > 1 {
				fmt.Println(match[1]) // 第二个元素包含第一个括号内的匹配内容
			} else {
				fmt.Println("No match found")
			}
		}
	})

}

func TestBracketRegex(t *testing.T) {
	fmt.Println(utils.StandardSongName("理想三旬（女声版） - 藤柒吖"))
	fmt.Println(utils.StandardSongName("小酒窝(Live) - 蔡卓妍 / 林俊杰"))
	fmt.Println(utils.StandardSongName("最后一页（完整版） - 洛尘鞅_"))
	fmt.Println(utils.StandardSongName("知我（抒情版） - 尘ah."))
	fmt.Println(utils.StandardSongName("幻听（女声版） - 星月酱"))
}

func TestDiscover(t *testing.T) {
	sample := []string{V1, V2, V3, V4, V5}
	for _, v := range sample {
		discover, err := NetEasyDiscover(v)
		assert.NoError(t, err)
		t.Log(discover)
	}
}
