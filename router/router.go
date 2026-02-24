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

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	{
		// 单曲相关
		api.GET("/search", handler.SearchMusic)
		api.GET("/parse", handler.ParseLink)
		api.POST("/download_url", handler.GetDownloadUrl)
		api.POST("/lyric", handler.GetLyric)

		// 歌单相关
		api.GET("/playlist/search", handler.SearchPlaylist)
		api.GET("/playlist/recommend", handler.GetRecommendPlaylists)
		api.GET("/playlist/parse", handler.ParsePlaylistLink)
		api.GET("/playlist/detail", handler.GetPlaylistDetail)
	}

	return r
}
