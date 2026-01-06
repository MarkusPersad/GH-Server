package middleware

import (
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/zaplog"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)
var(
		jwtSecret = []byte(os.Getenv("JWT_SECRET"))
		jwtExpire,_ = strconv.Atoi(os.Getenv("JWT_EXPIRE"))
)

type JwtClaim struct {
	UUID     uuid.UUID `json:"uuid"`
	UserName string    `json:"userName"`
	Email    string    `json:"email"`
	jwt.RegisteredClaims
}

func NewJwtMiddleWare() fiber.Handler{
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			Key: jwtSecret,
		},
		Extractor: extractors.FromAuthHeader("Bearer"),
		Claims: &JwtClaim{},
		ErrorHandler: func(ctx fiber.Ctx,err error) error{
			return exceptions.ErrInvalidToken
		},
		Next: func(c fiber.Ctx) bool {
			return strings.Contains(c.Request().URI().String(),"register")||
				strings.Contains(c.Request().URI().String(),"login")||
				strings.Contains(c.Request().URI().String(),"sendVerifyCode") ||
				strings.Contains(c.Request().URI().String(),"metrics")||
				strings.Contains(c.Request().URI().String(),"health")
		},
	})
}

func CreateJwtToken(user *model.User)(string,error){
	claims := JwtClaim{
		UUID: user.UUID,
		UserName: user.UserName,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour*24*time.Duration(jwtExpire))),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	tokenStr,err := token.SignedString(jwtSecret)
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Token String Generated Failed:%v",err))
		return "",err
	}
	return tokenStr,nil
}