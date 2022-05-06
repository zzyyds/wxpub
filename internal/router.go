package internal

import (
	"public-platform-manager/internal/domain/repository"
	"public-platform-manager/internal/infrastructure/persistence"
	"public-platform-manager/internal/interfaces/controller"
	"public-platform-manager/internal/interfaces/middleware"

	"github.com/gin-gonic/gin"
)

var (
	wx          *controller.WX
	accessToken *controller.AccessToken
	user        *controller.User
	template    *controller.Template
	msg         *controller.Message
)

func registerController() {
	wx = controller.NewWXController(repository.NewWXRepository())
	accessToken = controller.NewAccessTokenController(
		repository.NewAccessTokenRepository(
			persistence.NewAkRepo()))
	user = controller.NewUserController(repository.NewUserRepository())
	template = controller.NewTemplateController(
		repository.NewTemplateRepository(
			persistence.NewTemplateRepo()))
	msg = controller.NewMessageController(
		repository.NewMessageRepository(
			persistence.NewMessageRepo()))
}

func Run() *gin.Engine {
	registerController()
	engine := gin.Default()
	initRouter(engine)
	return engine
}

func initRouter(router *gin.Engine) {
	open := router.Group("/")
	routerWX(open)

	router.Use(middleware.NovaContext)
	// todo:鉴权认证
	interval := router.Group("/interval/v1")

	// access token
	routerAccessToken(interval)

	// 获取wx user info
	routerUser(interval)

	// 模板管理
	routerTemplate(interval)

	// 消息推送
	routerMsgPush(interval)
}

func routerWX(router *gin.RouterGroup) {
	wxGroup := router.Group("")
	{
		// wx开放平台接入测试接口
		wxGroup.GET("", wx.GetWXCheckSign)
		// todo: 暂时先用明文传输，后续补充aes加密传输
		// wx开放平台事件接收
		wxGroup.POST("", wx.GetEventXml)
	}
}

func routerAccessToken(router *gin.RouterGroup) {
	akGroup := router.Group("/access_token")
	{
		// 获取wx access token
		akGroup.GET("", accessToken.GetAccessToken)
		// 刷新wx access token
		akGroup.GET("/fresh", accessToken.FreshAccessToken)
	}
}

func routerUser(router *gin.RouterGroup) {
	userGroup := router.Group("/user")
	{
		userGroup.GET("", user.ListUser)
		userGroup.GET("/:id", user.GetUser)
	}
}

func routerTemplate(router *gin.RouterGroup) {
	templateGroup := router.Group("/template")
	{
		templateGroup.GET("", template.ListTemplate)
	}
}

func routerMsgPush(router *gin.RouterGroup) {
	msgPushGroup := router.Group("/message/push")
	{
		// 模板消息推送
		tmplSubGroup := msgPushGroup.Group("/tmpl")
		{
			tmplSubGroup.POST("", msg.SendTmplMessage)
		}
		// // 告警事件推送
		// alertSubGroup := msgPushGroup.Group("/alert")
		// {
		// 	alertSubGroup.GET("")
		// }
	}
}
