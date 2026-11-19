package handler

import (
	"testing"

	"github.com/guohuiyuan/music-lib/model"
)

func TestIsShareLink(t *testing.T) {
	tests := map[string]bool{
		"https://music.163.com/song?id=1": true,
		"http://example.com/song":         true,
		"spotify:track:abc":               true,
		"  SPOTIFY:album:abc  ":           true,
		"晴天":                              false,
		"spotify-search":                  false,
	}
	for input, want := range tests {
		if got := isShareLink(input); got != want {
			t.Errorf("isShareLink(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestSongExtension(t *testing.T) {
	tests := []struct {
		name   string
		ext    string
		rawURL string
		want   string
	}{
		{name: "explicit ext", ext: ".FLAC", rawURL: "https://example.com/song.mp3", want: "flac"},
		{name: "url ext", rawURL: "https://example.com/song.m4a?token=1", want: "m4a"},
		{name: "default", rawURL: "https://example.com/stream", want: "mp3"},
	}
	for _, test := range tests {
		song := &model.Song{Ext: test.ext}
		if got := songExtension(song, test.rawURL); got != test.want {
			t.Errorf("%s: songExtension() = %q, want %q", test.name, got, test.want)
		}
	}
}
