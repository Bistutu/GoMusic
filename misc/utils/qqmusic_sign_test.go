package utils

import (
	"strings"
	"testing"
)

func TestQQSignatureVectors(t *testing.T) {
	for input, want := range map[string]string{
		"":      "zzb98ffe087addcnuyjec90xpfdilcqea80d149dc",
		"hello": "zzbd4c1b979iwwqbnoib0krwpk3fitla7b14da20",
		"{\"req_0\":{\"id\":9625868647},\"title\":\"歌曲\"}": "zzb2f29bfc8ox6kwffmxiqjxqwzs9mlhg8f33fc1f",
	} {
		if got := Encrypt(input); got != want {
			t.Errorf("signature %q = %s want %s", input, got, want)
		}
		if strings.ContainsAny(Encrypt(input), "/+=") {
			t.Fatal("signature contains forbidden base64 characters")
		}
	}
}
