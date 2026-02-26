package handler

import (
	"net/http"

	"github.com/guohuiyuan/go-music-api/service"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/music-lib/model"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// SongReq 专用请求体，确保 Swagger 页面只显示这三个字段
type SongReq struct {
	ID     string            `json:"id" binding:"required" example:"31445554"`
	Source string            `json:"source" binding:"required" example:"netease"`
	Extra  map[string]string `json:"extra,omitempty" example:"{\"song_id\":\"31445554\"}"`
}

// @Summary 搜索单曲
// @Description 根据关键词和指定的平台搜索音乐
// @Tags 单曲功能
// @Param keyword query string true "搜索关键词" default(抖音)
// @Param source query string true "音乐平台(qq/netease/kuwo/kugou/migu/bilibili/fivesing/joox/soda/jamendo/qianqian等)" default(qq)
// @Produce json
// @Success 200 {object} Response
// @Router /api/search [get]
func SearchMusic(c *gin.Context) {
	keyword := c.Query("keyword")
	source := c.Query("source")

	if keyword == "" || source == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "参数缺失"})
		return
	}

	searchFn := service.GetSearchFunc(source)
	if searchFn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "不支持的源"})
		return
	}

	songs, err := searchFn(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}

	for i := range songs {
		songs[i].Source = source
	}

	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: songs})
}

// @Summary 解析单曲链接
// @Description 粘贴任意平台的单曲分享链接，自动识别平台并解析出歌曲详情
// @Tags 单曲功能
// @Param link query string true "单曲分享链接" default(https://y.qq.com/n/ryqq/songDetail/0039MnYb0qxYhV)
// @Produce json
// @Success 200 {object} Response
// @Router /api/parse [get]
func ParseLink(c *gin.Context) {
	link := c.Query("link")
	if link == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "链接不能为空"})
		return
	}

	source := service.DetectSource(link)
	if source == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无法识别链接所属平台"})
		return
	}

	parseFn := service.GetParseFunc(source)
	if parseFn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "该平台暂不支持解析链接"})
		return
	}

	song, err := parseFn(link)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "解析失败: " + err.Error()})
		return
	}
	song.Source = source

	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: song})
}

// @Summary 搜索歌单
// @Description 通过关键词搜索指定平台的歌单列表
// @Tags 歌单功能
// @Param keyword query string true "歌单关键词" default(华语流行)
// @Param source query string true "音乐平台" default(netease)
// @Produce json
// @Success 200 {object} Response
// @Router /api/playlist/search [get]
func SearchPlaylist(c *gin.Context) {
	keyword := c.Query("keyword")
	source := c.Query("source")
	if keyword == "" || source == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "参数缺失"})
		return
	}
	fn := service.GetPlaylistSearchFunc(source)
	if fn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "不支持该源搜索歌单"})
		return
	}
	playlists, err := fn(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}
	for i := range playlists {
		playlists[i].Source = source
	}
	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: playlists})
}

// @Summary 获取歌单详情(包含歌曲列表)
// @Description 根据歌单ID和指定的平台，获取歌单内的所有歌曲
// @Tags 歌单功能
// @Param id query string true "歌单ID" default(3778678)
// @Param source query string true "音乐平台" default(netease)
// @Produce json
// @Success 200 {object} Response
// @Router /api/playlist/detail [get]
func GetPlaylistDetail(c *gin.Context) {
	id := c.Query("id")
	source := c.Query("source")

	if id == "" || source == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "参数缺失"})
		return
	}

	fn := service.GetPlaylistDetailFunc(source)
	if fn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "不支持获取该源的歌单详情"})
		return
	}

	songs, err := fn(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}

	for i := range songs {
		songs[i].Source = source
	}

	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: songs})
}

// @Summary 每日推荐歌单
// @Description 获取指定平台的热门或每日推荐歌单
// @Tags 歌单功能
// @Param source query string true "音乐平台" default(qq)
// @Produce json
// @Success 200 {object} Response
// @Router /api/playlist/recommend [get]
func GetRecommendPlaylists(c *gin.Context) {
	source := c.Query("source")
	if source == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "未指定来源"})
		return
	}
	fn := service.GetRecommendFunc(source)
	if fn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "该平台暂不支持推荐"})
		return
	}
	playlists, err := fn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}
	for i := range playlists {
		playlists[i].Source = source
	}
	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: playlists})
}

// @Summary 解析歌单链接
// @Description 粘贴歌单分享链接，自动识别平台并解析出歌单详情及包含的所有歌曲
// @Tags 歌单功能
// @Param link query string true "歌单分享链接" default(https://music.163.com/#/playlist?id=3778678)
// @Produce json
// @Success 200 {object} Response
// @Router /api/playlist/parse [get]
func ParsePlaylistLink(c *gin.Context) {
	link := c.Query("link")
	source := service.DetectSource(link)
	if source == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无法识别链接"})
		return
	}
	fn := service.GetParsePlaylistFunc(source)
	if fn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "该平台暂不支持解析歌单"})
		return
	}
	playlist, songs, err := fn(link)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}
	for i := range songs {
		songs[i].Source = source
	}
	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: gin.H{
		"playlist": playlist,
		"songs":    songs,
	}})
}

// @Summary 获取下载/播放链接
// @Description 传入带源和ID的歌曲JSON获取底层音频直链 (通常作为播放地址使用)
// @Tags 核心解析
// @Accept json
// @Produce json
// @Param req body SongReq true "请求参数"
// @Success 200 {object} Response
// @Router /api/download_url [post]
func GetDownloadUrl(c *gin.Context) {
	var req SongReq
	// 1. 绑定到精简版的请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的请求参数: " + err.Error()})
		return
	}

	// 2. 构造底层需要的完整 model.Song
	song := &model.Song{
		ID:     req.ID,
		Source: req.Source,
		Extra:  req.Extra,
	}

	dlFn := service.GetDownloadFunc(song.Source)
	if dlFn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "不支持获取该平台下载链接"})
		return
	}

	url, err := dlFn(song)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "获取链接失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: gin.H{"url": url}})
}

// @Summary 获取歌曲歌词
// @Description 传入带源和ID的歌曲JSON获取LRC格式的歌词文本
// @Tags 核心解析
// @Accept json
// @Produce json
// @Param req body SongReq true "请求参数"
// @Success 200 {object} Response
// @Router /api/lyric [post]
func GetLyric(c *gin.Context) {
	var req SongReq
	// 1. 绑定到精简版的请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的请求参数: " + err.Error()})
		return
	}

	// 2. 构造底层需要的完整 model.Song
	song := &model.Song{
		ID:     req.ID,
		Source: req.Source,
		Extra:  req.Extra,
	}

	lyricFn := service.GetLyricFunc(song.Source)
	if lyricFn == nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "不支持获取该平台歌词"})
		return
	}

	lyric, err := lyricFn(song)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "获取歌词失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: gin.H{"lyric": lyric}})
}