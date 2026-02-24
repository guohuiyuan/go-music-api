// @title Go Music API
// @version 1.0
// @description 这是一个基于底层库构建的跨平台音乐搜索与解析统一 API 服务。
// @host localhost:8080
// @BasePath /
package main

import (
	"fmt"

	"github.com/guohuiyuan/go-music-api/router"
	"github.com/guohuiyuan/go-music-api/service"
)

func main() {
	service.CM.Load()
	fmt.Println("Cookies 已加载")

	r := router.SetupRouter()

	fmt.Println("Music API Server is running on http://localhost:8080")
	fmt.Println("Swagger API 接口文档请访问: http://localhost:8080/swagger/index.html")
	if err := r.Run(":8080"); err != nil {
		panic("Failed to start server: " + err.Error())
	}
}
