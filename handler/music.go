package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	data, err := os.ReadFile("cookies.json")
	if err != nil {
		c.JSON(200, gin.H{})
		return
	}
	var cookies map[string]string
	json.Unmarshal(data, &cookies)
	c.JSON(200, cookies)
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
		data, _ := json.MarshalIndent(req, "", "  ")
		_ = os.WriteFile("cookies.json", data, 0644)
		service.CM.Load()
		c.JSON(200, gin.H{"status": "ok"})
	} else {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
	}
}

// ==========================================
// 核心：统一搜索与链接解析接口
// ==========================================

// UnifiedSearch 综合搜索与链接解析
// @Summary 综合搜索与链接解析
// @Description 兼容多源并发搜索以及链接智能解析，自动返回单曲或歌单数组。支持直接输入关键词或粘贴音乐平台的分享链接。
// @Tags Music
// @Produce json
// @Param q query string true "关键词或音乐分享链接" default(香水有毒) example(香水有毒)
// @Param type query string false "搜索类型: song (单曲) 或 playlist (歌单)" Enums(song, playlist) default(song)
// @Param sources query []string false "指定的音源数组(留空则默认全平台)。例: netease, qq" collectionFormat(multi)
// @Success 200 {object} Response "成功时返回解析的数据，包含歌曲/歌单列表"
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
		if searchType == "playlist" {
			sources = []string{"netease", "qq", "kugou", "kuwo", "bilibili", "soda", "fivesing"}
		} else {
			sources = []string{"netease", "qq", "kugou", "kuwo", "bilibili", "migu", "soda", "fivesing"}
		}
	}

	var allSongs []model.Song
	var allPlaylists []model.Playlist
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
			errorMsg = fmt.Sprintf("解析失败: 暂不支持 %s 平台的此链接类型或解析出错", src)
		}
	} else {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, src := range sources {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				if searchType == "playlist" {
					if fn := service.GetPlaylistSearchFunc(s); fn != nil {
						if res, err := fn(keyword); err == nil {
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
	id := c.Query("id")
	source := c.Query("source")
	name := c.DefaultQuery("name", "Unknown")
	artist := c.DefaultQuery("artist", "Unknown")

	if id == "" || source == "" {
		c.String(400, "Missing params")
		return
	}

	tempSong := &model.Song{ID: id, Source: source, Name: name, Artist: artist}
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
	id := c.Query("id")
	src := c.Query("source")
	durStr := c.Query("duration")

	var urlStr string
	var err error

	if src == "soda" {
		cookie := service.CM.Get("soda")
		sodaInst := soda.New(cookie)
		info, sErr := sodaInst.GetDownloadInfo(&model.Song{ID: id, Source: src})
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
		urlStr, err = fn(&model.Song{ID: id, Source: src})
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
	id, src := c.Query("id"), c.Query("source")
	fn := service.GetDownloadFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "不支持的源"})
		return
	}
	urlStr, err := fn(&model.Song{ID: id, Source: src})
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
	id, src := c.Query("id"), c.Query("source")
	fn := service.GetLyricFunc(src)
	if fn == nil {
		c.JSON(400, Response{Code: 400, Msg: "无歌词支持"})
		return
	}
	lrc, _ := fn(&model.Song{ID: id, Source: src})
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
	id, src := c.Query("id"), c.Query("source")
	if fn := service.GetLyricFunc(src); fn != nil {
		if lrc, _ := fn(&model.Song{ID: id, Source: src}); lrc != "" {
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
	id, src := c.Query("id"), c.Query("source")
	name := c.DefaultQuery("name", "Unknown")
	artist := c.DefaultQuery("artist", "Unknown")

	fn := service.GetLyricFunc(src)
	if fn == nil {
		c.String(404, "No support")
		return
	}
	lrc, _ := fn(&model.Song{ID: id, Source: src})
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
		sources = []string{"netease", "qq", "kugou", "kuwo"} // 与 server.go 对齐
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