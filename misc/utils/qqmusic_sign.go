package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

func Encrypt(param string) string {
	k1 := map[string]int{
		"0": 0, "1": 1, "2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7, "8": 8, "9": 9,
		"A": 10, "B": 11, "C": 12, "D": 13, "E": 14, "F": 15,
	}
	l1 := []int{212, 45, 80, 68, 195, 163, 163, 203, 157, 220, 254, 91, 204, 79, 104, 6}
	t := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="

	//jsonData, _ := json.Marshal()
	md5Hash := md5.Sum([]byte(param))
	md5Str := strings.ToUpper(hex.EncodeToString(md5Hash[:]))

	t1 := selectChars(md5Str, []int{21, 4, 9, 26, 16, 20, 27, 30})
	t3 := selectChars(md5Str, []int{18, 11, 3, 2, 1, 7, 6, 25})

	ls2 := make([]int, 0, 16)
	for i := 0; i < 16; i++ {
		x1 := k1[string(md5Str[i*2])]
		x2 := k1[string(md5Str[i*2+1])]
		x3 := (x1*16 ^ x2) ^ l1[i]
		ls2 = append(ls2, x3)
	}

	ls3 := make([]string, 0, 7)
	for i := 0; i < 6; i++ {
		if i == 5 {
			ls3 = append(ls3, string(t[ls2[len(ls2)-1]>>2]), string(t[(ls2[len(ls2)-1]&3)<<4]))
		} else {
			x4 := ls2[i*3] >> 2
			x5 := (ls2[i*3+1] >> 4) ^ ((ls2[i*3] & 3) << 4)
			x6 := (ls2[i*3+2] >> 6) ^ ((ls2[i*3+1] & 15) << 2)
			x7 := 63 & ls2[i*3+2]
			ls3 = append(ls3, string(t[x4])+string(t[x5])+string(t[x6])+string(t[x7]))
		}
	}

	t2 := strings.Join(ls3, "")
	t2 = strings.ReplaceAll(t2, "[\\/+]", "")
	sign := "zzb" + strings.ToLower(t1+t2+t3)
	return sign
}

func selectChars(str string, indices []int) string {
	var result string
	for _, index := range indices {
		result += string(str[index])
	}
	return result
}
