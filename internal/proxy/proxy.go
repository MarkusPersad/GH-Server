package proxy

import (
	"GH-Server/pkg/zaplog"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/valyala/fasthttp"
)

var (
	tDToken = os.Getenv("TIANDITU_KEY")
)

func init() {
	proxy.WithClient(&fasthttp.Client{
    NoDefaultUserAgentHeader: true,
    DisablePathNormalizing:   true,
    // Allow self-signed certificates when proxying to HTTPS targets.
    TLSConfig: &tls.Config{
        	InsecureSkipVerify: true,
   		},
	})
}

// 天地图影像
func ImageryHandler(ctx fiber.Ctx) error {
	url := fmt.Sprintf("https://t1.tianditu.gov.cn/DataServer?T=img_w&x=%s&y=%s&l=%s&tk=%s",ctx.Params("x"),ctx.Params("y"),ctx.Params("z"),tDToken)
	if err := proxy.Do(ctx,url);err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Proxy failed: %v", err))
		return err
	} 
	return nil
}

// 天地图边界底图
func IboHandler(ctx fiber.Ctx) error {
	url := fmt.Sprintf("https://t%s.tianditu.gov.cn/DataServer?T=ibo_w&x=%s&y=%s&l=%s&tk=%s",ctx.Params("s"),ctx.Params("x"),ctx.Params("y"),ctx.Params("z"),tDToken) 
	if err := proxy.Do(ctx,url); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Proxy failed: %v", err))
		return err
	}
	return nil
}

// 天地图注记
func CiaHandler(ctx fiber.Ctx) error {
	url := fmt.Sprintf("https://t%s.tianditu.gov.cn/DataServer?T=cia_w&x=%s&y=%s&l=%s&tk=%s",ctx.Params("s"),ctx.Params("x"),ctx.Params("y"),ctx.Params("z"),tDToken)
	if err := proxy.Do(ctx,url); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Proxy failed: %v", err))
		return err
	}
	return nil
}

func ImageryMiddleware() fiber.Handler{
	return proxy.Balancer(proxy.Config{
		Servers: []string{
			"https://t0.tianditu.gov.cn/DataServer",
			"https://t1.tianditu.gov.cn/DataServer",
			"https://t2.tianditu.gov.cn/DataServer",
			"https://t3.tianditu.gov.cn/DataServer",
			"https://t4.tianditu.gov.cn/DataServer",
			"https://t5.tianditu.gov.cn/DataServer",
			"https://t6.tianditu.gov.cn/DataServer",
			"https://t7.tianditu.gov.cn/DataServer",
		},
		ModifyRequest: func(ctx fiber.Ctx) error {
			// ctx.Request().URI().QueryArgs().Add("T",ctx.Params("T"))
			// ctx.Request().URI().QueryArgs().Add("x",ctx.Params("x"))
			// ctx.Request().URI().QueryArgs().Add("y",ctx.Params("y"))
			// ctx.Request().URI().QueryArgs().Add("l",ctx.Params("z"))
			ctx.Request().URI().QueryArgs().Set("tk",tDToken)
			return nil
		},
		DialDualStack: true,
	})
}