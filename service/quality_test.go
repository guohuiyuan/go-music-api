package service

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/provider"
)

type fakeQualityProvider struct {
	urls  map[string]string
	calls []string
}

func (f *fakeQualityProvider) AvailableQualityOptions() ([]model.AudioQualityOption, error) {
	return nil, nil
}

func (f *fakeQualityProvider) GetDownloadURLWithQuality(_ *model.Song, quality string) (string, error) {
	f.calls = append(f.calls, quality)
	if downloadURL := f.urls[quality]; downloadURL != "" {
		return downloadURL, nil
	}
	return "", errors.New("quality unavailable")
}

var _ provider.AudioQualityProvider = (*fakeQualityProvider)(nil)

func TestResolveDownloadURLFallsBackToLowerTier(t *testing.T) {
	qualityProvider := &fakeQualityProvider{
		urls: map[string]string{
			"high": "https://example.com/song.mp3",
		},
	}
	song := &model.Song{Source: "netease", ID: "1"}
	resolved, err := resolveDownloadURL(qualityProvider, song, "master")
	if err != nil {
		t.Fatalf("resolveDownloadURL() error = %v", err)
	}
	if resolved.URL != "https://example.com/song.mp3" || resolved.Quality != "high" {
		t.Fatalf("resolved = %#v, want high-quality URL", resolved)
	}
	if len(qualityProvider.calls) < 2 || qualityProvider.calls[0] != "master" {
		t.Fatalf("fallback calls = %#v, want master followed by lower tiers", qualityProvider.calls)
	}
	if qualityProvider.calls[len(qualityProvider.calls)-1] != "high" {
		t.Fatalf("fallback calls = %#v, want to stop at high", qualityProvider.calls)
	}
}

func TestResolveDownloadURLHonorsSongQualityOverride(t *testing.T) {
	qualityProvider := &fakeQualityProvider{
		urls: map[string]string{
			"standard": "https://example.com/standard.mp3",
			"master":   "https://example.com/master.wav",
		},
	}
	song := &model.Song{
		Source: "netease",
		ID:     "1",
		Extra:  map[string]string{"quality": "standard"},
	}
	resolved, err := resolveDownloadURL(qualityProvider, song, "master")
	if err != nil {
		t.Fatalf("resolveDownloadURL() error = %v", err)
	}
	if resolved.Quality != "standard" {
		t.Fatalf("quality = %q, want standard", resolved.Quality)
	}
	if !reflect.DeepEqual(qualityProvider.calls, []string{"standard"}) {
		t.Fatalf("calls = %#v, want only standard", qualityProvider.calls)
	}
}

func TestResolveDownloadURLAutoUsesProviderDefault(t *testing.T) {
	qualityProvider := &fakeQualityProvider{
		urls: map[string]string{"auto": "https://example.com/auto.mp3"},
	}
	song := &model.Song{Source: "qq", ID: "1"}
	resolved, err := resolveDownloadURL(qualityProvider, song, "")
	if err != nil {
		t.Fatalf("resolveDownloadURL() error = %v", err)
	}
	if resolved.Quality != "auto" || !reflect.DeepEqual(qualityProvider.calls, []string{"auto"}) {
		t.Fatalf("resolved = %#v, calls = %#v", resolved, qualityProvider.calls)
	}
}

func TestAudioQualityCatalog(t *testing.T) {
	sources := GetAudioQualitySourceNames()
	if len(sources) == 0 {
		t.Fatal("quality source list is empty")
	}
	for _, source := range sources {
		options, err := GetAudioQualityChoices(source)
		if err != nil {
			t.Errorf("GetAudioQualityChoices(%q) error = %v", source, err)
			continue
		}
		if len(options) == 0 {
			t.Errorf("source %q has no quality options", source)
		}
		seen := make(map[model.AudioQuality]bool)
		for _, option := range options {
			if option.Value == "" {
				t.Errorf("source %q contains an empty quality value", source)
			}
			if seen[option.Value] {
				t.Errorf("source %q contains duplicate quality %q", source, option.Value)
			}
			seen[option.Value] = true
		}
		if audioQualityProviderForSource(source) == nil {
			t.Errorf("source %q has no account quality provider", source)
		}
	}
}

func TestAudioQualityFallbackRejectsUnsupportedQuality(t *testing.T) {
	song := &model.Song{Source: "netease", ID: "1"}
	_, err := audioQualityFallbackOrder(song, "eight-track")
	if err == nil || !strings.Contains(err.Error(), "unsupported audio quality") {
		t.Fatalf("error = %v, want unsupported quality error", err)
	}
}
