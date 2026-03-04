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
	url := fmt.Sprintf("https://t%s.tianditu.gov.cn/DataServer?T=img_w&x=%s&y=%s&l=%s&tk=%s",ctx.Params("s"),ctx.Params("x"),ctx.Params("y"),ctx.Params("z"),tDToken)
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