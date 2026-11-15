package logic

import (
	"GoMusic/misc/models"
	"testing"
)

func TestFormatTrackPreservesHyphens(t *testing.T) {
	track := models.Track{Title: "A - B（Live）【tag】", Artists: []string{"C - D", "E"}}
	if got := FormatTrack(track, Options{Format: "singer-song"}); got != "C - D / E - A - B (Live)" {
		t.Fatal(got)
	}
	if got := FormatTrack(track, Options{Format: "song", Detailed: true}); got != track.Title {
		t.Fatal(got)
	}
}
