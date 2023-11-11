package logic

import (
	"testing"
)

func TestRegex(t *testing.T) {
	// http://163cn.tv/zoIxm3
	// https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998
	// https://music.163.com/playlist?id=477577176&userid=341246998
	t.Log(netEasyV1Regex.MatchString("https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998"))
}
