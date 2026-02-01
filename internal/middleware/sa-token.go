package middleware

import (
	fibersatoken "GH-Server/internal/middleware/fiber-sa-token"
	"os"
	"strconv"

	"github.com/click33/sa-token-go/core"
	srds "github.com/click33/sa-token-go/storage/redis"
	rds "github.com/redis/go-redis/v9"
)

var (
	jwtSecret = os.Getenv("JWT_SECRET")
	tokenName = os.Getenv("SATOKEN_TOKENNAME")
	keyPrefix = os.Getenv("SATOKEN_KEYPREFIX")
	timeout,_   = strconv.Atoi(os.Getenv("SATOKEN_TIMEOUT"))
	isConcurrent,_ = strconv.ParseBool(os.Getenv("SATOKEN_ISCONCURRENT"))
	isShare,_ = strconv.ParseBool(os.Getenv("SATOKEN_ISSHARE"))
	maxLoginCount,_ = strconv.Atoi(os.Getenv("SATOKEN_MAXLOGINCOUNT"))
	isReadBody,_ = strconv.ParseBool(os.Getenv("SATOKEN_ISREADBODY"))
	isReadHeader,_ = strconv.ParseBool(os.Getenv("SATOKEN_ISREADHEADER"))
	isReadCookie,_ = strconv.ParseBool(os.Getenv("SATOKEN_ISREADCOOKIE"))
	tokenSessionCheckLogin,_ = strconv.ParseBool(os.Getenv("SATOKEN_TOKENSESSIONCHECKLOGIN"))
	autoRenew,_ = strconv.ParseBool(os.Getenv("SATOKEN_AUTORENEW"))
	isLog,_ = strconv.ParseBool(os.Getenv("SATOKEN_ISLOG")) 
)

func SaTokenMiddleware(rdb *rds.Client) {
	fibersatoken.SetManager(
        core.NewBuilder().
            Storage(srds.NewStorageFromClient(rdb)).
            TokenName(tokenName).
			JwtSecretKey(jwtSecret).
			MaxRefresh(core.DefaultConfig().MaxRefresh).
			RenewInterval(core.DefaultConfig().RenewInterval).
			ActiveTimeout(core.DefaultConfig().ActiveTimeout).
			IsConcurrent(isConcurrent).
			IsShare(isShare).
			MaxLoginCount(maxLoginCount).
			IsReadBody(isReadBody).
			IsReadHeader(isReadHeader).
			IsReadCookie(isReadCookie).
			DataRefreshPeriod(core.DefaultConfig().DataRefreshPeriod).
			TokenSessionCheckLogin(tokenSessionCheckLogin).
			AutoRenew(autoRenew).
			KeyPrefix(keyPrefix).
			IsLog(isLog).
            Timeout(int64(timeout)).                      
            TokenStyle(core.TokenStyleJWT). 
            IsPrintBanner(false).                
            Build(),
	)
}
