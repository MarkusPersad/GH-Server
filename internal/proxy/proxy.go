package proxy

import (
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/response"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"crypto/tls"
	"fmt"
	"net"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/valyala/fasthttp"
)

var (
	tDToken = os.Getenv("TIANDITU_KEY")
	tDServerToken = os.Getenv("TIANDITU_TOKEN")
	ipinfoToken = os.Getenv("IPINFO_TOKEN")
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
	url := fmt.Sprintf("https://t%s.tianditu.gov.cn/DataServer?T=%s&x=%s&y=%s&l=%s&tk=%s",ctx.Params("s"),ctx.Params("T"),ctx.Params("x"),ctx.Params("y"),ctx.Params("z"),tDToken)
	if err := proxy.Do(ctx,url);err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Proxy failed: %v", err))
		return err
	} 
	return nil
}

func GeoCoderHandler(ctx fiber.Ctx) error {
	request := new(request.GeoCoderRequest)
	if err := ctx.Bind().Body(request);err != nil {
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(request); err != nil {
		return exceptions.ErrInvalidParameters
	}
	url := fmt.Sprintf(`http://api.tianditu.gov.cn/geocoder?ds={"keyWord":"%s"}&tk=%s`,request.KeyWord,tDServerToken)
	zaplog.Zap.Info(fmt.Sprintf("Request URL: %s", url))
	if err := proxy.Do(ctx,url);err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Proxy failed: %v", err))
		return err
	}
	return nil
}

func IpInfoHandler(ctx fiber.Ctx) error {
	client := ipinfo.NewClient(nil,nil,ipinfoToken)
	ip := ctx.Params("ip")
	if ip == "" {
		return exceptions.ErrBadRequest
	}
	info,err := client.GetIPInfo(net.ParseIP(ip))
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get IP Info Error:%v",err.Error()))
		return exceptions.ErrInternalServerError
	}
	return ctx.Status(fiber.StatusOK).JSON(response.Success("IP获取成功",info)) 
}