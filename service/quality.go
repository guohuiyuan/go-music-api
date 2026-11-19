package service

import (
	"fmt"
	"strings"

	"github.com/guohuiyuan/music-lib/apple"
	"github.com/guohuiyuan/music-lib/bilibili"
	"github.com/guohuiyuan/music-lib/fivesing"
	"github.com/guohuiyuan/music-lib/jamendo"
	"github.com/guohuiyuan/music-lib/joox"
	"github.com/guohuiyuan/music-lib/kugou"
	"github.com/guohuiyuan/music-lib/kuwo"
	"github.com/guohuiyuan/music-lib/migu"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/netease"
	"github.com/guohuiyuan/music-lib/provider"
	"github.com/guohuiyuan/music-lib/qianqian"
	"github.com/guohuiyuan/music-lib/qq"
	"github.com/guohuiyuan/music-lib/soda"
	"github.com/guohuiyuan/music-lib/spotify"
)

type ResolvedAudioQuality struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

var audioQualitySourceOrder = []string{
	"netease",
	"qq",
	"kugou",
	"kuwo",
	"migu",
	"soda",
	"apple",
	"spotify",
	"qianqian",
	"joox",
	"bilibili",
	"fivesing",
	"jamendo",
}

var audioQualityChoices = map[string]func() []model.AudioQualityOption{
	"netease":  netease.ListQualityOptions,
	"qq":       qq.ListQualityOptions,
	"kugou":    kugou.ListQualityOptions,
	"kuwo":     kuwo.ListQualityOptions,
	"migu":     migu.ListQualityOptions,
	"soda":     soda.ListQualityOptions,
	"apple":    apple.ListQualityOptions,
	"spotify":  spotify.ListQualityOptions,
	"qianqian": qianqian.ListQualityOptions,
	"joox":     joox.ListQualityOptions,
	"bilibili": bilibili.ListQualityOptions,
	"fivesing": fivesing.ListQualityOptions,
	"jamendo":  jamendo.ListQualityOptions,
}

func GetAudioQualitySourceNames() []string {
	return append([]string(nil), audioQualitySourceOrder...)
}

func GetAudioQualityChoices(source string) ([]model.AudioQualityOption, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	factory, ok := audioQualityChoices[source]
	if !ok {
		return nil, fmt.Errorf("unsupported audio quality source: %s", source)
	}
	return append([]model.AudioQualityOption(nil), factory()...), nil
}

func GetAllAudioQualityChoices() map[string][]model.AudioQualityOption {
	result := make(map[string][]model.AudioQualityOption, len(audioQualitySourceOrder))
	for _, source := range audioQualitySourceOrder {
		options, _ := GetAudioQualityChoices(source)
		result[source] = options
	}
	return result
}

func GetAudioQualityOptions(source string) ([]model.AudioQualityOption, error) {
	qualityProvider := audioQualityProviderForSource(source)
	if qualityProvider == nil {
		return nil, fmt.Errorf("unsupported audio quality source: %s", strings.TrimSpace(source))
	}
	return qualityProvider.AvailableQualityOptions()
}

func GetDownloadFuncWithQuality(source, requestedQuality string) func(*model.Song) (string, error) {
	qualityProvider := audioQualityProviderForSource(source)
	if qualityProvider == nil {
		return nil
	}
	return func(song *model.Song) (string, error) {
		resolved, err := resolveDownloadURL(qualityProvider, song, requestedQuality)
		if err != nil {
			return "", err
		}
		return resolved.URL, nil
	}
}

func ResolveDownloadURL(source string, song *model.Song, requestedQuality string) (ResolvedAudioQuality, error) {
	qualityProvider := audioQualityProviderForSource(source)
	if qualityProvider == nil {
		return ResolvedAudioQuality{}, fmt.Errorf("unsupported audio quality source: %s", strings.TrimSpace(source))
	}
	return resolveDownloadURL(qualityProvider, song, requestedQuality)
}

func ResolveSodaDownloadInfo(song *model.Song, requestedQuality string) (*soda.DownloadInfo, string, error) {
	sodaProvider := soda.New(CM.Get("soda"))
	order, err := audioQualityFallbackOrder(song, requestedQuality)
	if err != nil {
		return nil, "", err
	}

	var lastErr error
	for _, quality := range order {
		info, resolveErr := sodaProvider.GetDownloadInfoWithQuality(song, string(quality))
		if resolveErr == nil && info != nil && strings.TrimSpace(info.URL) != "" {
			return info, string(quality), nil
		}
		if resolveErr != nil {
			lastErr = resolveErr
		} else {
			lastErr = fmt.Errorf("soda returned an empty download URL for quality %s", quality)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no playable soda quality was found")
	}
	return nil, "", fmt.Errorf("audio unavailable from quality %s downward: %w", order[0], lastErr)
}

func audioQualityProviderForSource(source string) provider.AudioQualityProvider {
	source = strings.ToLower(strings.TrimSpace(source))
	candidate := sourceProvider(source, CM.Get(source))
	if candidate == nil {
		return nil
	}
	qualityProvider, ok := candidate.(provider.AudioQualityProvider)
	if !ok {
		return nil
	}
	return qualityProvider
}

func resolveDownloadURL(
	qualityProvider provider.AudioQualityProvider,
	song *model.Song,
	requestedQuality string,
) (ResolvedAudioQuality, error) {
	order, err := audioQualityFallbackOrder(song, requestedQuality)
	if err != nil {
		return ResolvedAudioQuality{}, err
	}

	var lastErr error
	for _, quality := range order {
		downloadURL, resolveErr := qualityProvider.GetDownloadURLWithQuality(song, string(quality))
		if resolveErr == nil && strings.TrimSpace(downloadURL) != "" {
			return ResolvedAudioQuality{
				URL:     downloadURL,
				Quality: string(quality),
			}, nil
		}
		if resolveErr != nil {
			lastErr = resolveErr
		} else {
			lastErr = fmt.Errorf("source returned an empty download URL for quality %s", quality)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no playable quality was found")
	}
	return ResolvedAudioQuality{}, fmt.Errorf(
		"audio unavailable from quality %s downward: %w",
		order[0],
		lastErr,
	)
}

func audioQualityFallbackOrder(song *model.Song, requestedQuality string) ([]model.AudioQuality, error) {
	if song == nil {
		return nil, fmt.Errorf("song is required")
	}

	quality := resolveSongAudioQuality(song, requestedQuality)
	normalized, ok := model.NormalizeAudioQuality(quality)
	if !ok {
		return nil, fmt.Errorf("unsupported audio quality: %s", quality)
	}
	if normalized == model.AudioQualityAuto {
		return []model.AudioQuality{model.AudioQualityAuto}, nil
	}

	choices, err := GetAudioQualityChoices(song.Source)
	if err != nil {
		return nil, err
	}
	for index, option := range choices {
		if option.Value != normalized {
			continue
		}
		order := make([]model.AudioQuality, 0, len(choices)-index)
		for _, fallback := range choices[index:] {
			order = append(order, fallback.Value)
		}
		return order, nil
	}
	return nil, fmt.Errorf(
		"audio quality %s is not supported by source %s",
		normalized,
		song.Source,
	)
}

func resolveSongAudioQuality(song *model.Song, requestedQuality string) string {
	if song != nil && song.Extra != nil {
		if override := strings.TrimSpace(song.Extra["quality"]); override != "" {
			return override
		}
	}
	if requested := strings.TrimSpace(requestedQuality); requested != "" {
		return requested
	}
	return string(model.AudioQualityAuto)
}
