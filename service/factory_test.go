package service

import (
	"testing"
)

func TestFactorySourceCoverage(t *testing.T) {
	for _, source := range GetAllSourceNames() {
		if GetSearchFunc(source) == nil {
			t.Errorf("source %q has no song search", source)
		}
		if GetParseFunc(source) == nil {
			t.Errorf("source %q has no song parser", source)
		}
		if GetDownloadFunc(source) == nil {
			t.Errorf("source %q has no download resolver", source)
		}
		if GetLyricFunc(source) == nil {
			t.Errorf("source %q has no lyric provider", source)
		}
	}

	for _, source := range GetPlaylistSourceNames() {
		if GetPlaylistSearchFunc(source) == nil {
			t.Errorf("source %q has no playlist search", source)
		}
		if GetPlaylistDetailFunc(source) == nil {
			t.Errorf("source %q has no playlist detail", source)
		}
		if GetParsePlaylistFunc(source) == nil {
			t.Errorf("source %q has no playlist parser", source)
		}
	}

	for _, source := range GetAlbumSourceNames() {
		if GetAlbumSearchFunc(source) == nil {
			t.Errorf("source %q has no album search", source)
		}
		if GetAlbumDetailFunc(source) == nil {
			t.Errorf("source %q has no album detail", source)
		}
		if GetParseAlbumFunc(source) == nil {
			t.Errorf("source %q has no album parser", source)
		}
	}

	for _, source := range GetPlaylistCategorySourceNames() {
		if GetPlaylistCategoriesFunc(source) == nil {
			t.Errorf("source %q has no playlist categories", source)
		}
		if GetCategoryPlaylistsFunc(source) == nil {
			t.Errorf("source %q has no category playlists", source)
		}
	}

	for _, source := range GetRecommendSourceNames() {
		if GetRecommendFunc(source) == nil {
			t.Errorf("source %q has no recommended playlists", source)
		}
	}

	for _, source := range GetTopListSourceNames() {
		if GetTopListsFunc(source) == nil {
			t.Errorf("source %q has no top-list catalog", source)
		}
		if GetTopListSongsFunc(source) == nil {
			t.Errorf("source %q has no top-list songs", source)
		}
		if GetHotSongsFunc(source) == nil {
			t.Errorf("source %q has no hot songs", source)
		}
	}

	for _, source := range GetUserPlaylistSourceNames() {
		if GetUserPlaylistsFunc(source) == nil {
			t.Errorf("source %q has no user playlists", source)
		}
	}

	for _, source := range GetQRLoginSourceNames() {
		if GetQRLoginCreateFunc(source) == nil {
			t.Errorf("source %q has no QR login create function", source)
		}
		if GetQRLoginCheckFunc(source) == nil {
			t.Errorf("source %q has no QR login check function", source)
		}
	}
}

func TestDetectSourceSupportsNewProviders(t *testing.T) {
	tests := map[string]string{
		"https://music.apple.com/cn/album/example/123?i=456": "apple",
		"https://open.spotify.com/track/abc123":              "spotify",
		"spotify:track:abc123":                               "spotify",
	}
	for link, want := range tests {
		if got := DetectSource(link); got != want {
			t.Errorf("DetectSource(%q) = %q, want %q", link, got, want)
		}
	}
}

func TestDefaultSourcesIncludeNewPlayableProviders(t *testing.T) {
	sources := make(map[string]bool)
	for _, source := range GetDefaultSourceNames() {
		sources[source] = true
	}
	for _, source := range []string{"apple", "soda", "qianqian"} {
		if !sources[source] {
			t.Errorf("default sources do not include %q", source)
		}
	}
	if sources["spotify"] {
		t.Error("spotify should remain opt-in because it is not a direct playback source")
	}
}
