package musicurl

import "testing"

func TestParseKugouCodes(t *testing.T) {
	for _, tc := range []struct{ input, id string }{
		{"40725013", "40725013"},
		{" \n40725013\t", "40725013"},
		{"酷狗码：40725013", "40725013"},
		{"酷狗码 40725013", "40725013"},
		{"Kugou code: 40725013", "40725013"},
		{"酷狗码: 00725013", "00725013"},
		{"1", "1"},
		{"12345678901234567890", "12345678901234567890"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			link, err := Parse(tc.input)
			if err != nil || link.Provider != Kugou || link.Kind != "code" || link.ID != tc.id {
				t.Fatalf("got %#v, %v", link, err)
			}
		})
	}
}

func TestRejectMalformedKugouCodes(t *testing.T) {
	for _, input := range []string{
		"123456789012345678901", "40 725013", "+40725013", "40725013.0", "4e7", "４０７２５０１３",
		"酷狗码：40725013abc", "我的编号是40725013", "https://evil.test/?code=40725013", "酷狗码：https://evil.test/40725013",
	} {
		t.Run(input, func(t *testing.T) {
			if link, err := Parse(input); err == nil {
				t.Fatalf("accepted malformed input %#v", link)
			}
		})
	}
}

func TestOfficialShareLinkTakesPrecedenceOverCodeText(t *testing.T) {
	link, err := Parse("酷狗码：40725013 https://m.kugou.com/songlist/gcid_3zbivkavz4z072/")
	if err != nil || link.Kind != "gcid" || link.ID != "gcid_3zbivkavz4z072" {
		t.Fatalf("got %#v, %v", link, err)
	}
}
