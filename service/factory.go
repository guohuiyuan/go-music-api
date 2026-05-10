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

type SearchFunc func(keyword string) ([]model.Song, error)
type SearchPlaylistFunc func(keyword string) ([]model.Playlist, error)
type PlaylistCategoriesFunc func() ([]model.PlaylistCategory, error)
type CategoryPlaylistsFunc func(string, int, int) ([]model.Playlist, error)
type QRLoginCreateFunc func() (*model.QRLoginSession, error)
type QRLoginCheckFunc func(string) (*model.QRLoginResult, error)
type UserPlaylistsFunc func(page, limit int) ([]model.Playlist, error)

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

func (m *CookieManager) SetAll(cookies map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for source, cookie := range cookies {
		source = strings.TrimSpace(source)
		cookie = strings.TrimSpace(cookie)
		if source == "" {
			continue
		}
		if cookie == "" {
			delete(m.cookies, source)
			continue
		}
		m.cookies[source] = cookie
	}
}

func (m *CookieManager) GetAll() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string, len(m.cookies))
	for source, cookie := range m.cookies {
		result[source] = cookie
	}
	return result
}

func (m *CookieManager) Save() error {
	data, err := json.MarshalIndent(m.GetAll(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(CookieFile, data, 0644)
}

func DetectSource(link string) string {
	if strings.Contains(link, "163.com") {
		return "netease"
	}
	if strings.Contains(link, "qq.com") {
		return "qq"
	}
	if strings.Contains(link, "5sing") {
		return "fivesing"
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
	if strings.Contains(link, "joox.com") {
		return "joox"
	}
	if strings.Contains(link, "bilibili.com") || strings.Contains(link, "b23.tv") {
		return "bilibili"
	}
	if strings.Contains(link, "douyin.com") || strings.Contains(link, "qishui") {
		return "soda"
	}
	if strings.Contains(link, "91q.com") {
		return "qianqian"
	}
	if strings.Contains(link, "jamendo.com") {
		return "jamendo"
	}
	return ""
}

func GetAllSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "fivesing", "jamendo", "joox", "qianqian", "soda", "bilibili"}
}

func GetPlaylistSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "jamendo", "joox", "qianqian", "bilibili", "soda", "fivesing"}
}

func GetAlbumSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "jamendo", "joox", "qianqian", "soda"}
}

func GetPlaylistCategorySourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "qianqian", "joox"}
}

func GetDefaultSourceNames() []string {
	allSources := GetAllSourceNames()
	defaultSources := make([]string, 0, len(allSources))
	excluded := map[string]bool{"bilibili": true, "joox": true, "jamendo": true, "fivesing": true}
	for _, source := range allSources {
		if !excluded[source] {
			defaultSources = append(defaultSources, source)
		}
	}
	return defaultSources
}

func GetRecommendSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo"}
}

func GetQRLoginSourceNames() []string {
	return []string{"netease", "qq", "qq_wx", "kugou", "bilibili"}
}

func GetUserPlaylistSourceNames() []string {
	return []string{"netease", "qq", "kugou"}
}

func GetSourceDescription(source string) string {
	descriptions := map[string]string{
		"netease":  "网易云音乐",
		"qq":       "QQ音乐",
		"kugou":    "酷狗音乐",
		"kuwo":     "酷我音乐",
		"migu":     "咪咕音乐",
		"fivesing": "5sing",
		"jamendo":  "Jamendo (CC)",
		"joox":     "JOOX",
		"qianqian": "千千音乐",
		"soda":     "汽水音乐",
		"bilibili": "Bilibili",
		"qq_wx":    "QQ音乐(微信扫码)",
	}
	if desc, exists := descriptions[source]; exists {
		return desc
	}
	return "未知音乐源"
}

func GetSearchFunc(source string) SearchFunc {
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

func GetAlbumSearchFunc(source string) SearchPlaylistFunc {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).SearchAlbum
	case "qq":
		return qq.New(c).SearchAlbum
	case "kugou":
		return kugou.New(c).SearchAlbum
	case "kuwo":
		return kuwo.New(c).SearchAlbum
	case "migu":
		return migu.New(c).SearchAlbum
	case "jamendo":
		return jamendo.New(c).SearchAlbum
	case "joox":
		return joox.New(c).SearchAlbum
	case "qianqian":
		return qianqian.New(c).SearchAlbum
	case "soda":
		return soda.New(c).SearchAlbum
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
	case "joox":
		return joox.New(c).Parse
	case "qianqian":
		return qianqian.New(c).Parse
	default:
		return nil
	}
}

// --- 追加：歌单相关工厂函数 ---

func GetPlaylistSearchFunc(source string) SearchPlaylistFunc {
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
	case "jamendo":
		return jamendo.New(c).SearchPlaylist
	case "joox":
		return joox.New(c).SearchPlaylist
	case "qianqian":
		return qianqian.New(c).SearchPlaylist
	default:
		return nil
	}
}

func GetAlbumDetailFunc(source string) func(string) ([]model.Song, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetAlbumSongs
	case "qq":
		return qq.New(c).GetAlbumSongs
	case "kugou":
		return kugou.New(c).GetAlbumSongs
	case "kuwo":
		return kuwo.New(c).GetAlbumSongs
	case "migu":
		return migu.New(c).GetAlbumSongs
	case "jamendo":
		return jamendo.New(c).GetAlbumSongs
	case "joox":
		return joox.New(c).GetAlbumSongs
	case "qianqian":
		return qianqian.New(c).GetAlbumSongs
	case "soda":
		return soda.New(c).GetAlbumSongs
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
	case "jamendo":
		return jamendo.New(c).GetPlaylistSongs
	case "joox":
		return joox.New(c).GetPlaylistSongs
	case "qianqian":
		return qianqian.New(c).GetPlaylistSongs
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

func GetPlaylistCategoriesFunc(source string) PlaylistCategoriesFunc {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetPlaylistCategories
	case "qq":
		return qq.New(c).GetPlaylistCategories
	case "kugou":
		return kugou.New(c).GetPlaylistCategories
	case "kuwo":
		return kuwo.New(c).GetPlaylistCategories
	case "migu":
		return migu.New(c).GetPlaylistCategories
	case "joox":
		return joox.New(c).GetPlaylistCategories
	case "qianqian":
		return qianqian.New(c).GetPlaylistCategories
	default:
		return nil
	}
}

func GetCategoryPlaylistsFunc(source string) CategoryPlaylistsFunc {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetCategoryPlaylists
	case "qq":
		return qq.New(c).GetCategoryPlaylists
	case "kugou":
		return kugou.New(c).GetCategoryPlaylists
	case "kuwo":
		return kuwo.New(c).GetCategoryPlaylists
	case "migu":
		return migu.New(c).GetCategoryPlaylists
	case "joox":
		return joox.New(c).GetCategoryPlaylists
	case "qianqian":
		return qianqian.New(c).GetCategoryPlaylists
	default:
		return nil
	}
}

func GetQRLoginCreateFunc(source string) QRLoginCreateFunc {
	switch source {
	case "netease":
		return netease.CreateQRLogin
	case "qq":
		return qq.CreateQRLogin
	case "qq_wx":
		return qq.CreateWXQRLogin
	case "kugou":
		return kugou.CreateQRLogin
	case "bilibili":
		return bilibili.CreateQRLogin
	default:
		return nil
	}
}

func GetQRLoginCheckFunc(source string) QRLoginCheckFunc {
	switch source {
	case "netease":
		return netease.CheckQRLogin
	case "qq":
		return qq.CheckQRLogin
	case "qq_wx":
		return qq.CheckWXQRLogin
	case "kugou":
		return kugou.CheckQRLogin
	case "bilibili":
		return bilibili.CheckQRLogin
	default:
		return nil
	}
}

func GetUserPlaylistsFunc(source string) UserPlaylistsFunc {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).GetUserPlaylists
	case "qq":
		return qq.New(c).GetUserPlaylists
	case "kugou":
		return kugou.New(c).GetUserPlaylists
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
	case "migu":
		return migu.New(c).ParsePlaylist
	case "jamendo":
		return jamendo.New(c).ParsePlaylist
	case "joox":
		return joox.New(c).ParsePlaylist
	case "qianqian":
		return qianqian.New(c).ParsePlaylist
	default:
		return nil
	}
}

func GetParseAlbumFunc(source string) func(string) (*model.Playlist, []model.Song, error) {
	c := CM.Get(source)
	switch source {
	case "netease":
		return netease.New(c).ParseAlbum
	case "qq":
		return qq.New(c).ParseAlbum
	case "kugou":
		return kugou.New(c).ParseAlbum
	case "kuwo":
		return kuwo.New(c).ParseAlbum
	case "migu":
		return migu.New(c).ParseAlbum
	case "jamendo":
		return jamendo.New(c).ParseAlbum
	case "joox":
		return joox.New(c).ParseAlbum
	case "qianqian":
		return qianqian.New(c).ParseAlbum
	case "soda":
		return soda.New(c).ParseAlbum
	default:
		return nil
	}
}
