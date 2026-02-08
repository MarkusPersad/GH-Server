package utils

import (
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/zaplog"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var (
	jwtSecret = os.Getenv("JWT_SECRET")
	jwtexpir,_ = strconv.Atoi(os.Getenv("JWT_EXPIRE"))
	jwtrefresh,_ = strconv.Atoi(os.Getenv("JWT_REFRESH"))
	appName = os.Getenv("APP_NAME")
	access = "access:"
	refresh = "refresh:"
	blackList = "blacklist:"
	refreshHeader = "X-Refresh"
	accessHeader = "Authorization"
)

type AccessClaims struct {
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Role        string   `json:"role"`
}

func CreateAccessToken(uuid string,) (string,error) {
	claims := &AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: uuid,
			ExpiresAt:jwt.NewNumericDate(time.Now().Add(time.Duration(jwtexpir)*time.Minute)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ID:fmt.Sprintf("%s%s",access,uuid),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	return  token.SignedString([]byte(jwtSecret))
}
func CreateRefreshToken(ctx fiber.Ctx,redisClient *redis.Client ,uuid string,role string)(string,error){
	claims := &RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID: fmt.Sprintf("%s%s",refresh,uuid),
			Subject: uuid,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtrefresh*24)*time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	tokenStr,err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Create Refresh Token Error:%v",err))
		return "",err
	}
	
	if err := redisClient.SetNX(ctx,claims.ID,tokenStr,time.Duration(jwtrefresh*24)*time.Hour).Err();err != nil{
		zaplog.Zap.Error(fmt.Sprintf("Redis Error:%v",err))
		return "",err
	}
	return tokenStr,nil
}

func ParseRefreshToken(tokenStr string)(*jwt.Token,error){
	token,err := jwt.ParseWithClaims(tokenStr,&RefreshClaims{},func(t *jwt.Token) (any, error) {
		return []byte(jwtSecret),nil
	})
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Parse Refresh Token Error:%v",err))
		if errors.Is(err,jwt.ErrTokenExpired) {
			return nil,exceptions.ErrTokenExpired
		}
		return nil,exceptions.ErrInvalidToken
	}
	if !token.Valid{
		return nil,exceptions.ErrInvalidToken
	}
	return token,nil
}