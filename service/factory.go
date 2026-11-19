package service

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

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
type TopListsFunc func() ([]model.TopList, error)
type TopListSongsFunc func(string) ([]model.Song, error)
type HotSongsFunc func(int) ([]model.Song, error)
type QRLoginCreateFunc func() (*model.QRLoginSession, error)
type QRLoginCheckFunc func(string) (*model.QRLoginResult, error)
type UserPlaylistsFunc func(page, limit int) ([]model.Playlist, error)

func sourceProvider(source, cookie string) any {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "netease":
		return netease.New(cookie)
	case "qq":
		return qq.New(cookie)
	case "kugou":
		return kugou.New(cookie)
	case "kuwo":
		return kuwo.New(cookie)
	case "migu":
		return migu.New(cookie)
	case "soda":
		return soda.New(cookie)
	case "apple":
		return apple.New(cookie)
	case "spotify":
		return spotify.New(cookie)
	case "qianqian":
		return qianqian.New(cookie)
	case "joox":
		return joox.New(cookie)
	case "bilibili":
		return bilibili.New(cookie)
	case "fivesing":
		return fivesing.New(cookie)
	case "jamendo":
		return jamendo.New(cookie)
	default:
		return nil
	}
}

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
	if strings.Contains(link, "music.apple.com") || strings.Contains(link, "itunes.apple.com") {
		return "apple"
	}
	if strings.Contains(link, "open.spotify.com") || strings.HasPrefix(strings.TrimSpace(link), "spotify:") {
		return "spotify"
	}
	return ""
}

func GetAllSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "soda", "apple", "spotify", "qianqian", "joox", "bilibili", "fivesing", "jamendo"}
}

func GetPlaylistSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "jamendo", "joox", "qianqian", "bilibili", "soda", "fivesing", "apple", "spotify"}
}

func GetAlbumSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "jamendo", "joox", "qianqian", "soda", "apple", "spotify"}
}

func GetPlaylistCategorySourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "migu", "qianqian", "joox", "soda", "apple", "spotify"}
}

func GetTopListSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "soda", "migu", "apple", "spotify"}
}

func GetHotSongSourceNames() []string {
	return append([]string(nil), GetTopListSourceNames()...)
}

func GetDefaultSourceNames() []string {
	allSources := GetAllSourceNames()
	defaultSources := make([]string, 0, len(allSources))
	excluded := map[string]bool{"bilibili": true, "joox": true, "jamendo": true, "fivesing": true, "spotify": true}
	for _, source := range allSources {
		if !excluded[source] {
			defaultSources = append(defaultSources, source)
		}
	}
	return defaultSources
}

func GetRecommendSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "spotify"}
}

func GetQRLoginSourceNames() []string {
	return []string{"netease", "qq", "qq_wx", "kugou", "bilibili", "soda"}
}

func GetUserPlaylistSourceNames() []string {
	return []string{"netease", "qq", "kugou", "soda", "spotify"}
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
		"apple":    "Apple Music",
		"spotify":  "Spotify",
		"qq_wx":    "QQ音乐(微信扫码)",
	}
	if desc, exists := descriptions[source]; exists {
		return desc
	}
	return "未知音乐源"
}

func GetSearchFunc(source string) SearchFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.MusicProvider); ok {
		return p.Search
	}
	return nil
}

func GetAlbumSearchFunc(source string) SearchPlaylistFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.AlbumProvider); ok {
		return p.SearchAlbum
	}
	return nil
}

func GetDownloadFunc(source string) func(*model.Song) (string, error) {
	return GetDownloadFuncWithQuality(source, "")
}

func GetLyricFunc(source string) func(*model.Song) (string, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.LyricProvider); ok {
		return p.GetLyrics
	}
	return nil
}

func GetParseFunc(source string) func(string) (*model.Song, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.SongParser); ok {
		return p.Parse
	}
	return nil
}

// --- 追加：歌单相关工厂函数 ---

func GetPlaylistSearchFunc(source string) SearchPlaylistFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.PlaylistProvider); ok {
		return p.SearchPlaylist
	}
	return nil
}

func GetAlbumDetailFunc(source string) func(string) ([]model.Song, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.AlbumProvider); ok {
		return p.GetAlbumSongs
	}
	return nil
}

func GetPlaylistDetailFunc(source string) func(string) ([]model.Song, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.PlaylistProvider); ok {
		return p.GetPlaylistSongs
	}
	return nil
}

func GetRecommendFunc(source string) func() ([]model.Playlist, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.RecommendedPlaylistProvider); ok {
		return p.GetRecommendedPlaylists
	}
	return nil
}

func GetPlaylistCategoriesFunc(source string) PlaylistCategoriesFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.PlaylistCategoryProvider); ok {
		return p.GetPlaylistCategories
	}
	return nil
}

func GetCategoryPlaylistsFunc(source string) CategoryPlaylistsFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.PlaylistCategoryProvider); ok {
		return p.GetCategoryPlaylists
	}
	return nil
}

func GetTopListsFunc(source string) TopListsFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.TopListProvider); ok {
		return p.GetTopLists
	}
	return nil
}

func GetTopListSongsFunc(source string) TopListSongsFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.TopListProvider); ok {
		return p.GetTopListSongs
	}
	return nil
}

func GetHotSongsFunc(source string) HotSongsFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.HotSongProvider); ok {
		return p.GetHotSongs
	}
	return nil
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
	case "soda":
		return soda.CreateQRLogin
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
	case "soda":
		return soda.CheckQRLogin
	default:
		return nil
	}
}

func GetUserPlaylistsFunc(source string) UserPlaylistsFunc {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.UserPlaylistProvider); ok {
		return p.GetUserPlaylists
	}
	return nil
}

func GetParsePlaylistFunc(source string) func(string) (*model.Playlist, []model.Song, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.PlaylistProvider); ok {
		return p.ParsePlaylist
	}
	return nil
}

func GetParseAlbumFunc(source string) func(string) (*model.Playlist, []model.Song, error) {
	if p, ok := sourceProvider(source, CM.Get(source)).(provider.AlbumProvider); ok {
		return p.ParseAlbum
	}
	return nil
}
