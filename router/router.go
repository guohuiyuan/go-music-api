package router

import (
	"github.com/guohuiyuan/go-music-api/handler"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/guohuiyuan/go-music-api/docs"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 跨域处理 (对齐 server.go 的 corsMiddleware)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ==========================================
	// 标准化 API 路由 (推荐外部项目接入使用)
	// ==========================================
	api := r.Group("/api/v1")
	{
		// 1. 系统配置
		sys := api.Group("/system")
		{
			sys.GET("/cookies", handler.GetCookies)
			sys.POST("/cookies", handler.SetCookies)
		}

		// 2. 单曲相关 (Music)
		music := api.Group("/music")
		{
			music.GET("/search", handler.UnifiedSearch)         // 综合搜索(支持链接解析与多源)
			music.GET("/url", handler.GetMusicUrl)              // 获取音频直链
			music.GET("/stream", handler.StreamMusic)           // 代理音频流(含soda解密) / 下载音频
			music.GET("/inspect", handler.InspectMusic)         // 探测音频大小与码率
			music.GET("/switch", handler.SwitchSource)          // 智能切换可用音源
			music.GET("/lyric", handler.GetLyric)               // 获取 JSON 格式歌词
			music.GET("/lyric/file", handler.DownloadLyricFile) // 下载 .lrc 歌词文件
			music.GET("/cover", handler.ProxyCover)             // 代理/下载封面图防盗链
		}

		// 3. 歌单相关 (Playlist)
		playlist := api.Group("/playlist")
		{
			playlist.GET("/detail", handler.GetPlaylistDetail)
			playlist.GET("/recommend", handler.GetRecommendPlaylists)
		}
	}

	// ==========================================
	// 兼容 server.go 专属路由组
	// ==========================================
	// 改组路由完全模拟了原 server.go 暴露的接口路径，并复用上述增强版 handler。
	// 直接挂载即可无缝衔接原有的网页前端。
	compat := r.Group("/music")
	{
		compat.GET("/cookies", handler.GetCookies)
		compat.POST("/cookies", handler.SetCookies)
		
		compat.GET("/search", handler.UnifiedSearch)            // 对应 server.go 的 /search
		compat.GET("/playlist", handler.GetPlaylistDetail)      // 对应 server.go 的 /playlist
		compat.GET("/recommend", handler.GetRecommendPlaylists) // 对应 server.go 的 /recommend
		
		compat.GET("/inspect", handler.InspectMusic)            // 对应 server.go 的 /inspect
		compat.GET("/switch_source", handler.SwitchSource)      // 对应 server.go 的 /switch_source
		compat.GET("/download", handler.StreamMusic)            // 对应 server.go 的 /download
		compat.GET("/download_lrc", handler.DownloadLyricFile)  // 对应 server.go 的 /download_lrc
		compat.GET("/download_cover", handler.ProxyCover)       // 对应 server.go 的 /download_cover
		compat.GET("/lyric", handler.GetLyricText)              // 对应 server.go 的 /lyric (纯文本返回)
	}

	return r
}