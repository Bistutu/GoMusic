package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Kugou 结构体
type Kugou struct {
	Tid      string
	DataList []map[string]string
}

// NewKugou 初始化Kugou结构体
func NewKugou(tid string) *Kugou {
	return &Kugou{
		Tid:      tid,
		DataList: make([]map[string]string, 0),
	}
}

// KugouSignature 计算酷狗sign，此函数的具体实现需根据您的kugou_signature函数转换
func (k *Kugou) KugouSignature(url string) string {
	uri := strings.Split(url, "?")[1]
	uriList := strings.Split(uri, "&")
	sort.Strings(uriList) // 对参数进行排序
	uri = "OIlwieks28dk2k092lksi2UIkp" + strings.Join(uriList, "") + "OIlwieks28dk2k092lksi2UIkp"
	hash := md5.Sum([]byte(uri))
	return hex.EncodeToString(hash[:])
}

// KugouList 获取酷狗歌单列表
func (k *Kugou) KugouList() ([]map[string]string, error) {
	url := fmt.Sprintf("http://gatewayretry.kugou.com/v2/get_other_list_file?specialid=%s&need_sort=1&module=CloudMusic&clientver=11239&pagesize=300&specalidpgc=%s&userid=0&page=1&type=0&area_code=1&appid=1005", k.Tid, k.Tid)

	// 构建请求头
	header := http.Header{
		"User-Agent": []string{"Android9-AndroidPhone-11239-18-0-playlist-wifi"},
		"Host":       []string{"gatewayretry.kugou.com"},
		"X-Router":   []string{"pubsongscdn.kugou.com"},
		"Mid":        []string{"239526275778893399526700786998289824956"},
		"Dfid":       []string{"-"},
		"Clienttime": []string{strconv.Itoa(int(time.Now().Unix()))},
	}

	signature := k.KugouSignature(url)
	url += "&signature=" + signature

	// 发起HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
		if err != nil {
			return nil, err
		}

		var jsonData map[string]interface{}
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			return nil, err
		}

		// 解析并处理数据
		// ...

		return k.DataList, nil
	}

	return nil, fmt.Errorf("请求失败，状态码：%d", resp.StatusCode)
}

func main() {
	kugou := NewKugou("628671")
	dataList, err := kugou.KugouList()
	if err != nil {
		fmt.Println("获取歌单列表失败:", err)
		return
	}

	// 输出获取的歌单信息
	fmt.Println(dataList)
}
