package middleware

import (
	fibersatoken "GH-Server/internal/middleware/fiber-sa-token"
	"os"

	srds "github.com/click33/sa-token-go/storage/redis"
	rds "github.com/redis/go-redis/v9"
)

var (
	tokenName = os.Getenv("SATOKEN_TOKENNAME")
)

func SaTokenMiddleware(rdb *rds.Client){
	store := srds.NewStorageFromClient(rdb)
	conf :=  &fibersatoken.Config{
		JwtSecretKey: jwtSecret,
		TokenStyle: fibersatoken.TokenStyleJWT,
		TokenName: tokenName,
		Timeout: fibersatoken.DefaultConfig().Timeout,
		MaxRefresh: fibersatoken.DefaultConfig().MaxRefresh,
		RenewInterval: fibersatoken.DefaultConfig().RenewInterval,
		ActiveTimeout: fibersatoken.DefaultConfig().ActiveTimeout,
		IsConcurrent: false,
		IsShare: false,
		MaxLoginCount: 120,
		IsReadBody: false,
		IsReadHeader: true,
		IsReadCookie: false,
		DataRefreshPeriod: fibersatoken.DefaultConfig().DataRefreshPeriod,
		TokenSessionCheckLogin: true,
		AutoRenew: true,
		IsLog: false,
		IsPrintBanner: true,
		KeyPrefix: tokenName,
	}
	manager := fibersatoken.NewManager(store,conf)
	fibersatoken.SetManager(manager)
}