package logic

import (
	"log"
	"testing"
)

func TestRegex(t *testing.T) {
	log.Println(netEasyPattern.MatchString("https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998"))
}
