package service

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/guohuiyuan/music-lib/bilibili"
	"github.com/guohuiyuan/music-lib/fivesing"
	"github.com/guohuiyuan/music-lib/jamendo"
	"github.com/guohuiyuan/music-lib/joox"
	"github.com/guohuiyuan/music-lib/kugou"
	"github.com/guohuiyuan/music-lib/kuwo"
	"github.com/guohuiyuan/music-lib/migu"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/netease"
	"github.com/guohuiyuan/music-lib/qianqian"
	"github.com/guohuiyuan/music-lib/qq"
	"github.com/guohuiyuan/music-lib/soda"
)

const CookieFile = "cookies.json"

type CookieManager struct {
	mu      sync.RWMutex
	cookies map[string]string
}

var CM = &CookieManager{cookies: make(map[string]string)}

func (m *CookieManager) Load() {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := os.ReadFile(CookieFile)
	if err == nil {
		_ = json.Unmarshal(data, &m.cookies)
	}
}

func (m *CookieManager) Get(source string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cookies[source]
}

func DetectSource(link string) string {
	if strings.Contains(link, "163.com") {
		return "netease"
	}
	if strings.Contains(link, "qq.com") {
		return "qq"
	}
	if strings.Contains(link, "kugou.com") {
		return "kugou"
	}
	if strings.Contains(link, "kuwo.cn") {
		return "kuwo"
	}
	if strings.Contains(link, "migu.cn") {
		return "migu"
	}
	if strings.Contains(link, "bilibili.com") || strings.Contains(link, "b23.tv") {
		return "bilibili"
	}
	if strings.Contains(link, "douyin.com") || strings.Contains(link, "qishui") {
		return "soda"
	}
	if strings.Contains(link, "5sing") {
		return "fivesing"
	}
	if strings.Contains(link, "jamendo.com") {
		return "jamendo"
	}
	return ""
}

func GetSearchFunc(source string) func(string) ([]model.Song, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).Search
	case "qq":
		return qq.New(c).Search
	case "kugou":
		return kugou.New(c).Search
	case "kuwo":
		return kuwo.New(c).Search
	case "migu":
		return migu.New(c).Search
	case "soda":
		return soda.New(c).Search
	case "bilibili":
		return bilibili.New(c).Search
	case "fivesing":
		return fivesing.New(c).Search
	case "jamendo":
		return jamendo.New(c).Search
	case "joox":
		return joox.New(c).Search
	case "qianqian":
		return qianqian.New(c).Search
	default:
		return nil
	}
}

func GetDownloadFunc(source string) func(*model.Song) (string, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetDownloadURL
	case "qq":
		return qq.New(c).GetDownloadURL
	case "kugou":
		return kugou.New(c).GetDownloadURL
	case "kuwo":
		return kuwo.New(c).GetDownloadURL
	case "migu":
		return migu.New(c).GetDownloadURL
	case "soda":
		return soda.New(c).GetDownloadURL
	case "bilibili":
		return bilibili.New(c).GetDownloadURL
	case "fivesing":
		return fivesing.New(c).GetDownloadURL
	case "jamendo":
		return jamendo.New(c).GetDownloadURL
	case "joox":
		return joox.New(c).GetDownloadURL
	case "qianqian":
		return qianqian.New(c).GetDownloadURL
	default:
		return nil
	}
}

func GetLyricFunc(source string) func(*model.Song) (string, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetLyrics
	case "qq":
		return qq.New(c).GetLyrics
	case "kugou":
		return kugou.New(c).GetLyrics
	case "kuwo":
		return kuwo.New(c).GetLyrics
	case "migu":
		return migu.New(c).GetLyrics
	case "soda":
		return soda.New(c).GetLyrics
	case "bilibili":
		return bilibili.New(c).GetLyrics
	case "fivesing":
		return fivesing.New(c).GetLyrics
	case "jamendo":
		return jamendo.New(c).GetLyrics
	case "joox":
		return joox.New(c).GetLyrics
	case "qianqian":
		return qianqian.New(c).GetLyrics
	default:
		return nil
	}
}

func GetParseFunc(source string) func(string) (*model.Song, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).Parse
	case "qq":
		return qq.New(c).Parse
	case "kugou":
		return kugou.New(c).Parse
	case "kuwo":
		return kuwo.New(c).Parse
	case "migu":
		return migu.New(c).Parse
	case "soda":
		return soda.New(c).Parse
	case "bilibili":
		return bilibili.New(c).Parse
	case "fivesing":
		return fivesing.New(c).Parse
	case "jamendo":
		return jamendo.New(c).Parse
	default:
		return nil
	}
}

// --- 追加：歌单相关工厂函数 ---

func GetPlaylistSearchFunc(source string) func(string) ([]model.Playlist, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).SearchPlaylist
	case "qq":
		return qq.New(c).SearchPlaylist
	case "kugou":
		return kugou.New(c).SearchPlaylist
	case "kuwo":
		return kuwo.New(c).SearchPlaylist
	case "bilibili":
		return bilibili.New(c).SearchPlaylist
	case "soda":
		return soda.New(c).SearchPlaylist
	case "fivesing":
		return fivesing.New(c).SearchPlaylist
	case "migu":
		return migu.New(c).SearchPlaylist
	default:
		return nil
	}
}

func GetPlaylistDetailFunc(source string) func(string) ([]model.Song, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetPlaylistSongs
	case "qq":
		return qq.New(c).GetPlaylistSongs
	case "kugou":
		return kugou.New(c).GetPlaylistSongs
	case "kuwo":
		return kuwo.New(c).GetPlaylistSongs
	case "bilibili":
		return bilibili.New(c).GetPlaylistSongs
	case "soda":
		return soda.New(c).GetPlaylistSongs
	case "fivesing":
		return fivesing.New(c).GetPlaylistSongs
	case "migu":
		return migu.New(c).GetPlaylistSongs
	default:
		return nil
	}
}

func GetRecommendFunc(source string) func() ([]model.Playlist, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetRecommendedPlaylists
	case "qq":
		return qq.New(c).GetRecommendedPlaylists
	case "kugou":
		return kugou.New(c).GetRecommendedPlaylists
	case "kuwo":
		return kuwo.New(c).GetRecommendedPlaylists
	default:
		return nil
	}
}

func GetParsePlaylistFunc(source string) func(string) (*model.Playlist, []model.Song, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).ParsePlaylist
	case "qq":
		return qq.New(c).ParsePlaylist
	case "kugou":
		return kugou.New(c).ParsePlaylist
	case "kuwo":
		return kuwo.New(c).ParsePlaylist
	case "bilibili":
		return bilibili.New(c).ParsePlaylist
	case "soda":
		return soda.New(c).ParsePlaylist
	case "fivesing":
		return fivesing.New(c).ParsePlaylist
	default:
		return nil
	}
}
