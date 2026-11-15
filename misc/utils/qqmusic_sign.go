package utils

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// Encrypt returns the public QQ Music zzb request signature, without a JavaScript VM.
func Encrypt(payload string) string {
	digest := md5.Sum([]byte(payload))
	hash := strings.ToUpper(hex.EncodeToString(digest[:]))
	mask := [16]byte{212, 45, 80, 68, 195, 163, 163, 203, 157, 220, 254, 91, 204, 79, 104, 6}
	var encoded [16]byte
	for i := range encoded {
		encoded[i] = digest[i] ^ mask[i]
	}
	middle := strings.NewReplacer("/", "", "+", "").Replace(base64.RawStdEncoding.EncodeToString(encoded[:]))
	return "zzb" + strings.ToLower(selectChars(hash, []int{21, 4, 9, 26, 16, 20, 27, 30})+middle+selectChars(hash, []int{18, 11, 3, 2, 1, 7, 6, 25}))
}
func selectChars(value string, indices []int) string {
	result := make([]byte, len(indices))
	for i, index := range indices {
		result[i] = value[index]
	}
	return string(result)
}
