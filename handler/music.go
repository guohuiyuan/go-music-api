package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/go-music-api/service"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/soda"
	"github.com/guohuiyuan/music-lib/utils"
)

// Response 统一响应结构体
type Response struct {
	Code int         `json:"code" example:"200"`
	Msg  string      `json:"msg" example:"success"`
	Data interface{} `json:"data,omitempty"`
}

const (
	UA_Common    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
	UA_Mobile    = "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1"
	Ref_Netease  = "http://music.163.com/"
	Ref_Bilibili = "https://www.bilibili.com/"
	Ref_Migu     = "http://music.migu.cn/"
)

// 辅助函数：构造带有 Cookie 和防盗链的 Request
func buildReq(method, urlStr, source, rangeHeader string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	req.Header.Set("User-Agent", UA_Common)
	if source == "bilibili" {
		req.Header.Set("Referer", Ref_Bilibili)
	} else if source == "netease" {
		req.Header.Set("Referer", Ref_Netease)
	} else if source == "migu" {
		req.Header.Set("User-Agent", UA_Mobile)
		req.Header.Set("Referer", Ref_Migu)
	} else if source == "qq" {
		req.Header.Set("Referer", "http://y.qq.com")
	}

	if cookie := service.CM.Get(source); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	return req, nil
}

// 辅助函数：设置文件下载 Header
func setDownloadHeader(c *gin.Context, filename string) {
	encoded := url.QueryEscape(filename)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=utf-8''%s", encoded, encoded))
}

func parseSongExtraQuery(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "null" {
		return nil
	}
	var direct map[string]string
	if err := json.Unmarshal([]byte(raw), &direct); err == nil {
		return direct
	}
	var generic map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return nil
	}
	extra := make(map[string]string, len(generic))
	for key, value := range generic {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			extra[key] = v
		case float64, bool:
			extra[key] = fmt.Sprint(v)
		default:
			if data, err := json.Marshal(v); err == nil {
				extra[key] = string(data)
			}
		}
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func songFromQuery(c *gin.Context) *model.Song {
	duration, _ := strconv.Atoi(strings.TrimSpace(c.Query("duration")))
	return &model.Song{
		ID:       strings.TrimSpace(c.Query("id")),
		Source:   strings.TrimSpace(c.Query("source")),
		Name:     strings.TrimSpace(c.Query("name")),
		Artist:   strings.TrimSpace(c.Query("artist")),
		Album:    strings.TrimSpace(c.Query("album")),
		Cover:    strings.TrimSpace(c.Query("cover")),
		Duration: duration,
		Extra:    parseSongExtraQuery(c.Query("extra")),
	}
}

func parsePositiveIntQuery(c *gin.Context, name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(c.Query(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// ==========================================
// 系统配置相关接口
// ==========================================

// GetCookies 获取当前系统配置的 Cookies
// @Summary 获取当前系统加载的 Cookies
// @Description 读取并在 JSON 格式下返回当前系统已配置的各平台 Cookies。
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string "成功返回各平台 Cookie 键值对"
// @Router /api/v1/system/cookies [get]
func GetCookies(c *gin.Context) {
	c.JSON(200, service.CM.GetAll())
}

// SetCookies 设置系统 Cookies
// @Summary 设置系统 Cookies
// @Description 接收 JSON 格式的平台 cookie 键值对，覆盖并保存到系统，实时生效。
// @Tags System
// @Accept json
// @Produce json
// @Param cookies body map[string]string true "平台Cookies映射示例：{\"netease\": \"os=pc;\", \"qq\": \"...\"}"
// @Success 200 {object} Response "操作成功"
// @Failure 400 {object} Response "参数解析失败"
// @Router /api/v1/system/cookies [post]
func SetCookies(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err == nil {
		service.CM.SetAll(req)
		if err := service.CM.Save(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	} else {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
	}
}

func qrLoginCookieString(result *model.QRLoginResult) string {
	if result == nil {
		return ""
	}
	if cookie := strings.TrimSpace(result.Cookie); cookie != "" {
		return cookie
	}
	if len(result.Cookies) == 0 {
		return ""
	}
	keys := make([]string, 0, len(result.Cookies))
	for key := range result.Cookies {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(result.Cookies[key])
		if value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	return strings.Join(parts, "; ")
}

func qrLoginCookieSource(source string) string {
	if source == "qq_wx" {
		return "qq"
	}
	return source
}

// GetQRLoginSources 获取支持扫码登录的平台
// @Summary 获取支持扫码登录的平台
// @Description 返回当前 API 支持创建二维码登录会话的平台列表。
// @Tags System
// @Produce json
// @Success 200 {object} Response "支持扫码登录的平台"
// @Router /api/v1/system/qr_login/sources [get]
func GetQRLoginSources(c *gin.Context) {
	sources := service.GetQRLoginSourceNames()
	data := make([]gin.H, 0, len(sources))
	for _, source := range sources {
		data = append(data, gin.H{
			"source": source,
			"name":   service.GetSourceDescription(source),
		})
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: data})
}

// CreateQRLogin 创建扫码登录会话
// @Summary 创建扫码登录会话
// @Description 为指定平台创建扫码登录会话，返回二维码 URL、二维码图片地址或平台登录 key。
// @Tags System
// @Produce json
// @Param source path string true "扫码登录平台" Enums(netease,qq,qq_wx,kugou,bilibili) example(qq_wx)
// @Success 200 {object} Response "扫码登录会话"
// @Failure 404 {object} Response "平台不支持扫码登录"
// @Router /api/v1/system/qr_login/{source} [post]
func CreateQRLogin(c *gin.Context) {
	source := strings.TrimSpace(c.Param("source"))
	fn := service.GetQRLoginCreateFunc(source)
	if fn == nil {
		c.JSON(404, Response{Code: 404, Msg: "unsupported qr login source"})
		return
	}
	session, err := fn()
	if err != nil {
		c.JSON(502, Response{Code: 502, Msg: err.Error()})
		return
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: session})
}

// CheckQRLogin 轮询扫码登录状态
// @Summary 轮询扫码登录状态
// @Description 使用创建扫码登录会话返回的 key 轮询登录状态；成功时自动写入 cookies.json。
// @Tags System
// @Produce json
// @Param source path string true "扫码登录平台" Enums(netease,qq,qq_wx,kugou,bilibili) example(qq_wx)
// @Param key query string true "扫码登录 key"
// @Success 200 {object} Response "扫码登录状态"
// @Failure 400 {object} Response "缺少 key"
// @Failure 404 {object} Response "平台不支持扫码登录"
// @Router /api/v1/system/qr_login/{source} [get]
func CheckQRLogin(c *gin.Context) {
	source := strings.TrimSpace(c.Param("source"))
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(400, Response{Code: 400, Msg: "missing qr login key"})
		return
	}
	fn := service.GetQRLoginCheckFunc(source)
	if fn == nil {
		c.JSON(404, Response{Code: 404, Msg: "unsupported qr login source"})
		return
	}
	result, err := fn(key)
	if err != nil {
		c.JSON(502, Response{Code: 502, Msg: err.Error()})
		return
	}
	if result != nil && result.Status == model.QRLoginStatusSuccess {
		cookie := qrLoginCookieString(result)
		if cookie != "" {
			cookieSource := qrLoginCookieSource(source)
			result.Cookie = cookie
			service.CM.SetAll(map[string]string{cookieSource: cookie})
			if err := service.CM.Save(); err == nil {
				if result.Extra == nil {
					result.Extra = make(map[string]string)
				}
				result.Extra["cookie_saved"] = "true"
				result.Extra["cookie_source"] = cookieSource
				result.Extra["cookie_length"] = strconv.Itoa(len(cookie))
			}
		}
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: result})
}

// ==========================================
// 核心：统一搜索与链接解析接口
// ==========================================

// UnifiedSearch 综合搜索与链接解析
// @Summary 综合搜索与链接解析
// @Description 兼容多源并发搜索以及链接智能解析，自动返回单曲、歌单或专辑数组。支持直接输入关键词或粘贴音乐平台的分享链接。
// @Tags Music
// @Produce json
// @Param q query string true "关键词或音乐分享链接" default(香水有毒) example(香水有毒)
// @Param type query string false "搜索类型: song (单曲)、playlist (歌单) 或 album (专辑)" Enums(song, playlist, album) default(song)
// @Param sources query []string false "指定的音源数组(留空则默认全平台)。例: netease, qq" collectionFormat(multi)
// @Success 200 {object} Response "成功时返回解析的数据，包含歌曲、歌单或专辑列表"
// @Failure 400 {object} Response "不支持的链接解析"
// @Failure 500 {object} Response "解析过程出现错误"
// @Router /api/v1/music/search [get]
func UnifiedSearch(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword")) // 兼容旧版
	}
	searchType := c.DefaultQuery("type", "song")
	sources := c.QueryArray("sources")

	if len(sources) == 0 {
		if searchType == "album" {
			sources = service.GetAlbumSourceNames()
		} else if searchType == "playlist" {
			sources = service.GetPlaylistSourceNames()
		} else {
			sources = service.GetDefaultSourceNames()
		}
	}

	var allSongs []model.Song
	var allPlaylists []model.Playlist
	var allAlbums []model.Playlist
	var errorMsg string

	if strings.HasPrefix(keyword, "http") {
		src := service.DetectSource(keyword)
		if src == "" {
			c.JSON(400, Response{Code: 400, Msg: "不支持该链接的解析，或无法识别来源"})
			return
		}

		parsed := false
		if parseFn := service.GetParseFunc(src); parseFn != nil {
			if song, err := parseFn(keyword); err == nil {
				allSongs = append(allSongs, *song)
				searchType = "song"
				parsed = true
			}
		}
		if !parsed {
			if parsePlaylistFn := service.GetParsePlaylistFunc(src); parsePlaylistFn != nil {
				if playlist, songs, err := parsePlaylistFn(keyword); err == nil {
					if searchType == "playlist" {
						allPlaylists = append(allPlaylists, *playlist)
					} else {
						allSongs = append(allSongs, songs...)
						searchType = "song"
					}
					parsed = true
				}
			}
		}
		if !parsed {
			if parseAlbumFn := service.GetParseAlbumFunc(src); parseAlbumFn != nil {
				if album, songs, err := parseAlbumFn(keyword); err == nil {
					if searchType == "album" {
						allAlbums = append(allAlbums, *album)
					} else {
						allSongs = append(allSongs, songs...)
						searchType = "song"
					}
					parsed = true
				}
			}
		}
		if !parsed {
			errorMsg = fmt.Sprintf("解析失败: 暂不支持 %s 平台的此链接类型或解析出错", src)
		}
	} else {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, src := range sources {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				if searchType == "album" {
					if fn := service.GetAlbumSearchFunc(s); fn != nil {
						if res, err := fn(keyword); err == nil {
							for i := range res {
								res[i].Source = s
							}
							mu.Lock()
							allAlbums = append(allAlbums, res...)
							mu.Unlock()
						}
					}
				} else if searchType == "playlist" {
					if fn := service.GetPlaylistSearchFunc(s); fn != nil {
						if res, err := fn(keyword); err == nil {
							for i := range res {
								res[i].Source = s
							}
							mu.Lock()
							allPlaylists = append(allPlaylists, res...)
							mu.Unlock()
						}
					}
				} else {
					if fn := service.GetSearchFunc(s); fn != nil {
						if res, err := fn(keyword); err == nil {
							for i := range res {
								res[i].Source = s
							}
							mu.Lock()
							allSongs = append(allSongs, res...)
							mu.Unlock()
						}
					}
				}
			}(src)
		}
		wg.Wait()
	}

	if errorMsg != "" {
		c.JSON(500, Response{Code: 500, Msg: errorMsg})
		return
	}

	c.JSON(200, Response{
		Code: 200,
		Msg:  "success",
		Data: gin.H{
			"type":      searchType,
			"songs":     allSongs,
			"playlists": allPlaylists,
			"albums":    allAlbums,
		},
	})
}

// ==========================================
// 单曲相关接口
// ==========================================

// StreamMusic 串流代理与下载音频
// @Summary 串流代理与下载音频
// @Description 包含完整的各平台流代理逻辑（解决跨域防盗链），并特殊支持 Soda(汽水音乐) 加密流数据的后端解密。
// @Tags Music
// @Produce audio/mpeg
// @Param id query string true "音乐 ID" default(240479) example(240479)
// @Param source query string true "音乐来源平台" Enums(netease, qq, kugou, kuwo, bilibili, soda, migu, fivesing) default(netease) example(netease)
// @Param name query string false "音乐名称 (用于生成下载文件名)" default(香水有毒) example(香水有毒)
// @Param artist query string false "歌手名称 (用于生成下载文件名)" default(胡杨林) example(胡杨林)
// @Success 200 {file} file "直接返回音频二进制流，支持 HTTP Range"
// @Failure 400 {string} string "参数缺失或非法"
// @Failure 404 {string} string "找不到音频URL"
// @Failure 500 {string} string "音频解密失败"
// @Router /api/v1/music/stream [get]
func StreamMusic(c *gin.Context) {
	tempSong := songFromQuery(c)
	id := tempSong.ID
	source := tempSong.Source
	name := tempSong.Name
	artist := tempSong.Artist
	if name == "" {
		name = "Unknown"
		tempSong.Name = name
	}
	if artist == "" {
		artist = "Unknown"
		tempSong.Artist = artist
	}

	if id == "" || source == "" {
		c.String(400, "Missing params")
		return
	}

	filename := fmt.Sprintf("%s - %s.mp3", name, artist)

	if source == "soda" {
		cookie := service.CM.Get("soda")
		sodaInst := soda.New(cookie)
		info, err := sodaInst.GetDownloadInfo(tempSong)
		if err != nil {
			c.String(502, "Soda info error")
			return
		}
		req, err := buildReq("GET", info.URL, "soda", "")
		if err != nil {
			c.String(502, "Soda request error")
			return
		}
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			c.String(502, "Soda stream error")
			return
		}
		defer resp.Body.Close()
		encryptedData, _ := io.ReadAll(resp.Body)
		finalData, err := soda.DecryptAudio(encryptedData, info.PlayAuth)
		if err != nil {
			c.String(500, "Decrypt failed")
			return
		}
		setDownloadHeader(c, filename)
		http.ServeContent(c.Writer, c.Request, filename, time.Now(), bytes.NewReader(finalData))
		return
	}

	dlFunc := service.GetDownloadFunc(source)
	if dlFunc == nil {
		c.String(400, "Unknown source")
		return
	}
	downloadUrl, err := dlFunc(tempSong)
	if err != nil || downloadUrl == "" {
		c.String(404, "Failed to get URL")
		return
	}

	req, err := buildReq("GET", downloadUrl, source, c.GetHeader("Range"))
	if err != nil {
		c.String(502, "Upstream request error")
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(502, "Upstream stream error")
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		if k != "Transfer-Encoding" && k != "Date" && k != "Access-Control-Allow-Origin" {
			c.Writer.Header()[k] = v
		}
	}

	setDownloadHeader(c, filename)
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// InspectMusic 探测音频大小与码率
// @Summary 探测音频大小与码率
// @Description 快速探测音频直链的可访问性，并根据 `Content-Range` 推算文件大小及大概码率。
// @Tags Music
// @Produce json
// @Param id query string true "音乐 ID" default(240479) example(240479)
// @Param source query string true "音乐来源平台" default(netease) example(netease)
// @Param duration query string false "音乐时长(秒)，提供可精确预估码率(kbps)" default(290) example(290)
// @Success 200 {object} Response "包含有效状态、真实URL、文件大小和码率等探测信息"
// @Router /api/v1/music/inspect [get]
func InspectMusic(c *gin.Context) {
	song := songFromQuery(c)
	src := song.Source
	durStr := c.Query("duration")

	var urlStr string
	var err error

	if src == "soda" {
		cookie := service.CM.Get("soda")
		sodaInst := soda.New(cookie)
		info, sErr := sodaInst.GetDownloadInfo(song)
		if sErr != nil {
			c.JSON(200, gin.H{"valid": false})
			return
		}
		urlStr = info.URL
	} else {
		fn := service.GetDownloadFunc(src)
		if fn == nil {
			c.JSON(200, gin.H{"valid": false})
			return
		}
		urlStr, err = fn(song)
		if err != nil || urlStr == "" {
			c.JSON(200, gin.H{"valid": false})
			return
		}
	}

	req, _ := buildReq("GET", urlStr, src, "bytes=0-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	valid := false
	var size int64 = 0

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 || resp.StatusCode == 206 {
			valid = true
			cr := resp.Header.Get("Content-Range")
			if parts := strings.Split(cr, "/"); len(parts) == 2 {
				size, _ = strconv.ParseInt(parts[1], 10, 64)
			} else {
				size = resp.ContentLength
			}
		}
	}

	bitrate := "-"
	if valid && size > 0 {
		dur, _ := strconv.Atoi(durStr)
		if dur > 0 {
			kbps := int((size * 8) / int64(dur) / 1000)
			bitrate = fmt.Sprintf("%d kbps", kbps)
		}
	}

	c.JSON(200, gin.H{
		"valid":   valid,
		"url":     urlStr,
		"size":    fmt.Sprintf("%.1f MB", float64(size)/1024/1024),
		"bitrate": bitrate,
	})
}

// SwitchSource 智能切换音源
// @Summary 智能切换可用的平替音源
// @Description 当某一平台的歌曲灰掉（无版权）时，智能寻源切换到其他存在该歌曲的可用平台。
// @Tags Music
// @Produce json
// @Param name query string true "歌曲名称 (非常关键的匹配项)" default(香水有毒) example(香水有毒)
// @Param artist query string false "歌手名称" default(胡杨林) example(胡杨林)
// @Param source query string true "当前损坏的音源(将跳过此源搜索)" default(netease) example(netease)
// @Param target query string false "指定目标尝试的音源，为空则遍历主流平台搜索" default() example()
// @Param duration query string false "原音频时长(秒)，提供此时长可极大提高匹配准确度" default(290) example(290)
// @Success 200 {object} model.Song "成功找到高匹配度的可用歌曲"
// @Failure 400 {object} Response "参数错误(缺失歌名)"
// @Failure 404 {object} Response "未匹配到任何可用平替源"
// @Router /api/v1/music/switch [get]
func SwitchSource(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	artist := strings.TrimSpace(c.Query("artist"))
	current := strings.TrimSpace(c.Query("source"))
	target := strings.TrimSpace(c.Query("target"))
	durationStr := strings.TrimSpace(c.Query("duration"))

	origDuration, _ := strconv.Atoi(durationStr)

	if name == "" {
		c.JSON(400, gin.H{"error": "missing name"})
		return
	}

	keyword := name
	if artist != "" {
		keyword = name + " " + artist
	}

	var sources []string
	if target != "" {
		sources = []string{target}
	} else {
		sources = []string{"netease", "qq", "kugou", "kuwo", "migu", "bilibili"}
	}

	type candidate struct {
		song    model.Song
		score   float64
		durDiff int
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var candidates []candidate

	for _, src := range sources {
		if src == "" || src == current || src == "soda" || src == "fivesing" {
			continue
		}
		fn := service.GetSearchFunc(src)
		if fn == nil {
			continue
		}

		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			res, err := fn(keyword)
			if (err != nil || len(res) == 0) && artist != "" {
				res, _ = fn(name)
			}
			if len(res) == 0 {
				return
			}

			limit := len(res)
			if limit > 8 {
				limit = 8
			}

			for i := 0; i < limit; i++ {
				cand := res[i]
				cand.Source = s
				score := calcSongSimilarity(name, artist, cand.Name, cand.Artist)
				if score <= 0 {
					continue
				}

				durDiff := 0
				if origDuration > 0 && cand.Duration > 0 {
					durDiff = intAbs(origDuration - cand.Duration)
					if !isDurationClose(origDuration, cand.Duration) {
						continue
					}
				}

				mu.Lock()
				candidates = append(candidates, candidate{song: cand, score: score, durDiff: durDiff})
				mu.Unlock()
			}
		}(src)
	}
	wg.Wait()

	if len(candidates) == 0 {
		c.JSON(404, gin.H{"error": "no match"})
		return
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].durDiff < candidates[j].durDiff
		}
		return candidates[i].score > candidates[j].score
	})

	var selected *model.Song
	var selectedScore float64
	for _, cand := range candidates {
		if validatePlayable(&cand.song) {
			tmp := cand.song
			selected = &tmp
			selectedScore = cand.score
			break
		}
	}
	if selected == nil {
		c.JSON(404, gin.H{"error": "no playable match"})
		return
	}

	c.JSON(200, gin.H{
		"id":       selected.ID,
		"name":     selected.Name,
		"artist":   selected.Artist,
		"album":    selected.Album,
		"duration": selected.Duration,
		"source":   selected.Source,
		"cover":    selected.Cover,
		"score":    selectedScore,
		"link":     selected.Link,
	})
}

// GetMusicUrl 辅助 API：获取音频裸直链
// @Summary 获取音频裸直链
// @Description 获取解析到的原始音频播放链接。注：部分平台需要客户端带上特定的防盗链 header。
// @Tags Music
// @Produce json
// @Param id query string true "音乐 ID" default(240479) example(240479)
// @Param source query string true "平台源" default(netease) example(netease)
// @Success 200 {object} Response "直接返回带有 url 的数据实体"
// @Failure 400 {object} Response "源不支持"
// @Failure 500 {object} Response "链接抓取失败"
// @Router /api/v1/music/url [get]
func GetMusicUrl(c *gin.Context) {
	song := songFromQuery(c)
	src := song.Source
	fn := service.GetDownloadFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "不支持的源"})
		return
	}
	urlStr, err := fn(song)
	if err != nil {
		c.JSON(500, Response{Code: 500, Msg: err.Error()})
		return
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: gin.H{"url": urlStr}})
}

// ==========================================
// 歌词与封面
// ==========================================

// GetLyric 获取 JSON 格式歌词
// @Summary 获取 JSON 格式歌词
// @Description 抓取对应歌曲的完整 LRC 歌词文本，以 JSON 格式返回。
// @Tags Music
// @Produce json
// @Param id query string true "音乐 ID" default(240479) example(240479)
// @Param source query string true "平台" default(netease) example(netease)
// @Success 200 {object} Response "包含 lyric 字符串属性的数据对象"
// @Failure 400 {object} Response "对应平台未实现歌词抓取"
// @Router /api/v1/music/lyric [get]
func GetLyric(c *gin.Context) {
	song := songFromQuery(c)
	src := song.Source
	fn := service.GetLyricFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "无歌词支持"})
		return
	}
	lrc, _ := fn(song)
	c.JSON(200, Response{Code: 200, Msg: "success", Data: gin.H{"lyric": lrc}})
}

// GetLyricText 返回纯文本歌词
// @Summary 返回纯文本歌词 (旧版兼容)
// @Description 直接返回 `text/plain` 格式的纯歌词内容。若拉取失败，返回默认占位符提示。
// @Tags Music (Compat)
// @Produce text/plain
// @Param id query string true "音乐 ID" default(240479) example(240479)
// @Param source query string true "平台" default(netease) example(netease)
// @Success 200 {string} string "LRC 文本"
// @Router /music/lyric [get]
func GetLyricText(c *gin.Context) {
	song := songFromQuery(c)
	src := song.Source
	if fn := service.GetLyricFunc(src); fn != nil {
		if lrc, _ := fn(song); lrc != "" {
			c.String(200, lrc)
			return
		}
	}
	c.String(200, "[00:00.00] 暂无歌词")
}

// DownloadLyricFile 下载 LRC 文件
// @Summary 下载 LRC 歌词文件
// @Description 作为附件直接下载 `.lrc` 后缀的歌词文件到本地。
// @Tags Music
// @Produce application/octet-stream
// @Param id query string true "音乐 ID" default(240479) example(240479)
// @Param source query string true "平台" default(netease) example(netease)
// @Param name query string false "音乐名称 (生成保存文件名)" default(香水有毒) example(香水有毒)
// @Param artist query string false "歌手名称 (生成保存文件名)" default(胡杨林) example(胡杨林)
// @Success 200 {file} file "纯文本文件流"
// @Router /api/v1/music/lyric/file [get]
func DownloadLyricFile(c *gin.Context) {
	song := songFromQuery(c)
	src := song.Source
	name := song.Name
	artist := song.Artist
	if name == "" {
		name = "Unknown"
	}
	if artist == "" {
		artist = "Unknown"
	}

	fn := service.GetLyricFunc(src)
	if fn == nil {
		c.String(404, "No support")
		return
	}
	lrc, _ := fn(song)
	if lrc == "" {
		c.String(404, "Lyric not found")
		return
	}

	setDownloadHeader(c, fmt.Sprintf("%s - %s.lrc", name, artist))
	c.String(200, lrc)
}

// ProxyCover 代理并下载封面防盗链
// @Summary 代理请求并下载封面图
// @Description 发送带伪造标头的请求拉取远端封面大图，避开网易云、QQ 音乐的图片防盗链 403 问题。
// @Tags Music
// @Produce image/jpeg
// @Param url query string true "封面图原始 URL (需经过 urlencode)" default(https://p1.music.126.net/u9YkzGKeL6VgHQZ1Zb-7Sw==/2529976256655220.jpg) example(https://p1.music.126.net/u9YkzGKeL6VgHQZ1Zb-7Sw==/2529976256655220.jpg)
// @Param name query string false "歌曲名(用于生成下载文件名)" default(香水有毒) example(香水有毒)
// @Param artist query string false "歌手名(用于生成下载文件名)" default(胡杨林) example(胡杨林)
// @Success 200 {file} file "原封不动的图片流"
// @Router /api/v1/music/cover [get]
func ProxyCover(c *gin.Context) {
	u := c.Query("url")
	if u == "" {
		return
	}
	resp, err := utils.Get(u, utils.WithHeader("User-Agent", UA_Common))
	if err == nil {
		setDownloadHeader(c, fmt.Sprintf("%s - %s.jpg", c.Query("name"), c.Query("artist")))
		c.Data(200, "image/jpeg", resp)
	}
}

// ==========================================
// 歌单相关接口
// ==========================================

// GetPlaylistDetail 获取歌单详情
// @Summary 获取歌单详情
// @Description 传入源平台的对应歌单 ID，全量拉取并返回歌单内的全部单曲列表。
// @Tags Playlist
// @Produce json
// @Param id query string true "歌单的内部 ID" default(596729952) example(596729952)
// @Param source query string true "歌单所属平台" default(netease) example(netease)
// @Success 200 {object} Response "成功的数组列表"
// @Failure 400 {object} Response "源不支持"
// @Router /api/v1/playlist/detail [get]
func GetPlaylistDetail(c *gin.Context) {
	id, src := c.Query("id"), c.Query("source")
	if id == "" || src == "" {
		c.JSON(400, Response{Code: 400, Msg: "参数缺失"})
		return
	}
	fn := service.GetPlaylistDetailFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "不支持获取该源的歌单"})
		return
	}
	songs, err := fn(id)
	if err != nil {
		c.JSON(500, Response{Code: 500, Msg: err.Error()})
		return
	}
	for i := range songs {
		songs[i].Source = src
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: songs})
}

// GetRecommendPlaylists 每日推荐歌单
// @Summary 获取每日推荐热门歌单
// @Description 异步并发调用所勾选平台的接口，聚合返回他们各自首页推荐的当红歌单数据。
// @Tags Playlist
// @Produce json
// @Param sources query []string false "要获取的推荐平台列表 (留空则使用默认配置)" collectionFormat(multi) default(netease,qq,kugou,kuwo)
// @Success 200 {object} Response "各个平台的推荐歌单数组"
// @Router /api/v1/playlist/recommend [get]
func GetRecommendPlaylists(c *gin.Context) {
	sources := c.QueryArray("sources")
	if len(sources) == 0 {
		sources = service.GetRecommendSourceNames()
	}

	var allPlaylists []model.Playlist
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, src := range sources {
		fn := service.GetRecommendFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			res, err := fn()
			if err == nil && len(res) > 0 {
				for i := range res {
					res[i].Source = s
				}
				mu.Lock()
				allPlaylists = append(allPlaylists, res...)
				mu.Unlock()
			}
		}(src)
	}
	wg.Wait()
	c.JSON(200, Response{Code: 200, Msg: "success", Data: allPlaylists})
}

// GetAlbumDetail 获取专辑详情
// @Summary 获取专辑详情
// @Description 传入源平台的专辑 ID，返回专辑内歌曲列表。
// @Tags Album
// @Produce json
// @Param id query string true "专辑 ID" example(12345)
// @Param source query string true "专辑所属平台" Enums(netease,qq,kugou,kuwo,migu,jamendo,joox,qianqian,soda) default(netease)
// @Success 200 {object} Response "专辑歌曲列表"
// @Failure 400 {object} Response "源不支持或参数缺失"
// @Router /api/v1/album/detail [get]
func GetAlbumDetail(c *gin.Context) {
	id, src := strings.TrimSpace(c.Query("id")), strings.TrimSpace(c.Query("source"))
	if id == "" || src == "" {
		c.JSON(400, Response{Code: 400, Msg: "参数缺失"})
		return
	}
	fn := service.GetAlbumDetailFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "不支持获取该源的专辑"})
		return
	}
	songs, err := fn(id)
	if err != nil {
		c.JSON(500, Response{Code: 500, Msg: err.Error()})
		return
	}
	for i := range songs {
		songs[i].Source = src
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: songs})
}

// GetPlaylistCategories 获取歌单分类
// @Summary 获取歌单分类
// @Description 获取一个或多个平台支持的歌单分类标签。
// @Tags Playlist
// @Produce json
// @Param sources query []string false "指定平台列表，留空则使用全部支持分类的平台" collectionFormat(multi)
// @Success 200 {object} Response "按平台分组的歌单分类"
// @Router /api/v1/playlist/categories [get]
func GetPlaylistCategories(c *gin.Context) {
	sources := c.QueryArray("sources")
	if len(sources) == 0 {
		sources = service.GetPlaylistCategorySourceNames()
	}
	type categorySource struct {
		Source     string                   `json:"source"`
		Name       string                   `json:"name"`
		Categories []model.PlaylistCategory `json:"categories"`
		Error      string                   `json:"error,omitempty"`
	}
	results := make([]categorySource, 0, len(sources))
	for _, src := range sources {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		item := categorySource{Source: src, Name: service.GetSourceDescription(src)}
		fn := service.GetPlaylistCategoriesFunc(src)
		if fn == nil {
			item.Error = "unsupported source"
			results = append(results, item)
			continue
		}
		categories, err := fn()
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Categories = categories
		}
		results = append(results, item)
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: results})
}

// GetCategoryPlaylists 获取分类歌单
// @Summary 获取分类歌单
// @Description 按平台和分类 ID 分页获取歌单。
// @Tags Playlist
// @Produce json
// @Param source query string true "平台" Enums(netease,qq,kugou,kuwo,migu,qianqian,joox) default(netease)
// @Param category_id query string true "分类 ID"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(30)
// @Success 200 {object} Response "分类歌单列表"
// @Failure 400 {object} Response "源不支持或参数缺失"
// @Router /api/v1/playlist/category [get]
func GetCategoryPlaylists(c *gin.Context) {
	src := strings.TrimSpace(c.Query("source"))
	categoryID := strings.TrimSpace(c.Query("category_id"))
	if categoryID == "" {
		categoryID = strings.TrimSpace(c.Query("id"))
	}
	page := parsePositiveIntQuery(c, "page", 1)
	limit := parsePositiveIntQuery(c, "limit", 30)
	if src == "" || categoryID == "" {
		c.JSON(400, Response{Code: 400, Msg: "参数缺失"})
		return
	}
	fn := service.GetCategoryPlaylistsFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "不支持获取该源的分类歌单"})
		return
	}
	playlists, err := fn(categoryID, page, limit)
	if err != nil {
		c.JSON(500, Response{Code: 500, Msg: err.Error()})
		return
	}
	for i := range playlists {
		playlists[i].Source = src
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: gin.H{
		"source":      src,
		"category_id": categoryID,
		"page":        page,
		"limit":       limit,
		"playlists":   playlists,
	}})
}

// GetUserPlaylists 获取个人歌单
// @Summary 获取个人歌单
// @Description 使用已配置 Cookie 获取登录账号的个人歌单。QQ 支持我喜欢、个人目录歌单和收藏歌单。
// @Tags Playlist
// @Produce json
// @Param source query string true "平台" Enums(netease,qq,kugou) default(qq)
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(30)
// @Success 200 {object} Response "个人歌单列表"
// @Failure 400 {object} Response "源不支持或参数缺失"
// @Router /api/v1/playlist/user [get]
func GetUserPlaylists(c *gin.Context) {
	src := strings.TrimSpace(c.Query("source"))
	page := parsePositiveIntQuery(c, "page", 1)
	limit := parsePositiveIntQuery(c, "limit", 30)
	if src == "" {
		c.JSON(400, Response{Code: 400, Msg: "参数缺失"})
		return
	}
	fn := service.GetUserPlaylistsFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "不支持获取该源的个人歌单"})
		return
	}
	playlists, err := fn(page, limit)
	if err != nil {
		status := 500
		if strings.Contains(strings.ToLower(err.Error()), "require cookie") {
			status = 401
		}
		c.JSON(status, Response{Code: status, Msg: err.Error()})
		return
	}
	for i := range playlists {
		playlists[i].Source = src
	}
	c.JSON(200, Response{Code: 200, Msg: "success", Data: gin.H{
		"source":    src,
		"page":      page,
		"limit":     limit,
		"playlists": playlists,
	}})
}

// ==========================================
// 算法与校验辅助函数 (用于 SwitchSource)
// ==========================================

func validatePlayable(song *model.Song) bool {
	if song == nil || song.ID == "" || song.Source == "" {
		return false
	}
	if song.Source == "soda" || song.Source == "fivesing" {
		return false
	}
	fn := service.GetDownloadFunc(song.Source)
	if fn == nil {
		return false
	}
	urlStr, err := fn(&model.Song{ID: song.ID, Source: song.Source})
	if err != nil || urlStr == "" {
		return false
	}
	req, err := buildReq("GET", urlStr, song.Source, "bytes=0-1")
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 || resp.StatusCode == 206
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func isDurationClose(a, b int) bool {
	if a <= 0 || b <= 0 {
		return true
	}
	diff := intAbs(a - b)
	if diff <= 10 {
		return true
	}
	maxAllowed := int(float64(a) * 0.15)
	if maxAllowed < 10 {
		maxAllowed = 10
	}
	return diff <= maxAllowed
}

func calcSongSimilarity(name, artist, candName, candArtist string) float64 {
	nameA := normalizeText(name)
	nameB := normalizeText(candName)
	if nameA == "" || nameB == "" {
		return 0
	}
	nameSim := similarityScore(nameA, nameB)

	artistA := normalizeText(artist)
	artistB := normalizeText(candArtist)
	if artistA == "" || artistB == "" {
		return nameSim
	}
	artistSim := similarityScore(artistA, artistB)
	return nameSim*0.7 + artistSim*0.3
}

func normalizeText(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.In(r, unicode.Han) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func similarityScore(a, b string) float64 {
	if a == b {
		return 1
	}
	if a == "" || b == "" {
		return 0
	}
	la := len([]rune(a))
	lb := len([]rune(b))
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 0
	}
	dist := levenshteinDistance(a, b)
	if dist >= maxLen {
		return 0
	}
	return 1 - float64(dist)/float64(maxLen)
}

func levenshteinDistance(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = del
			if ins < cur[j] {
				cur[j] = ins
			}
			if sub < cur[j] {
				cur[j] = sub
			}
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}
