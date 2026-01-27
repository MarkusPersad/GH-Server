package middleware

import (
	fibersatoken "GH-Server/internal/middleware/fiber-sa-token"
	srds "github.com/click33/sa-token-go/storage/redis"
	rds "github.com/redis/go-redis/v9"
)

func SaTokenMiddleware(rdb *rds.Client){
	store := srds.NewStorageFromClient(rdb)
	conf := fibersatoken.DefaultConfig()
	manager := fibersatoken.NewManager(store,conf)
	fibersatoken.SetManager(manager)
}