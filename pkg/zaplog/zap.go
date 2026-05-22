package zaplog

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/utils/v2"
	utilsstrings "github.com/gofiber/utils/v2/strings"
	_ "github.com/joho/godotenv/autoload" // 确保尽早自动加载 .env
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New 创建一个整合了 Lumberjack 轮转功能的 zap 日志中间件
func New(config ...Config) fiber.Handler {
	// 1. 动态从环境变量读取日志轮转配置（整合自原 log.go 逻辑）
	logFile := os.Getenv("LOG_FILE")
	logLevel := os.Getenv("LOG_LEVEL")
	logFormat := os.Getenv("LOG_TIME_FORMAT")
	logMaxSize, _ := strconv.ParseInt(os.Getenv("LOG_MAX_SIZE"), 10, 64)
	logMaxBackups, _ := strconv.ParseInt(os.Getenv("LOG_MAX_BACKUPS"), 10, 64)
	logMaxAge, _ := strconv.ParseInt(os.Getenv("LOG_MAX_AGE"), 10, 64)
	logCompress, _ := strconv.ParseBool(os.Getenv("LOG_COMPRESS"))

	// 2. 组装 Zap 编码器与 Lumberjack 轮转核心
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "Time",
		LevelKey:       "Level",
		NameKey:        "Log",
		CallerKey:      "Caller",
		MessageKey:     "Message",
		StacktraceKey:  "StackTrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			if logFormat == "" {
				logFormat = "2006-01-02 15:04:05"
			}
			enc.AppendString(t.Format(logFormat))
		},
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxAge:     int(logMaxAge),
		MaxBackups: int(logMaxBackups),
		MaxSize:    int(logMaxSize),
		Compress:   logCompress,
	}

	// 解析日志级别
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel // 解析失败默认 info
	}

	// 针对 debug 级别开启控制台双写，否则仅写入轮转文件
	var writerSync zapcore.WriteSyncer
	if strings.ToLower(logLevel) == "debug" {
		writerSync = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter))
	} else {
		writerSync = zapcore.AddSync(fileWriter)
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writerSync,
		level,
	)
	
	// 构建专属于该中间件的具有轮转功能的 Logger 实例
	rotationLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// 3. 应用中间件的基本配置
	cfg := configDefault(config...)
	
	// 核心修复点：将配置中的 Logger 替换为我们刚刚动态构建的带轮转功能的 rotationLogger
	cfg.Logger = rotationLogger

	// 核心修复点：动态调整默认的 Levels 级别，防止正常请求的日志因全局级别过高而在 Check 阶段被吞掉
	// 如果全局只允许 warn 以上，则将成功的日志级别也强行标记为 warn 级别发送，确保能够调用 Write 触发轮转
	if len(cfg.Levels) >= 3 {
		if !rotationLogger.Core().Enabled(zapcore.InfoLevel) && rotationLogger.Core().Enabled(zapcore.WarnLevel) {
			cfg.Levels[2] = zapcore.WarnLevel
		} else if !rotationLogger.Core().Enabled(zapcore.WarnLevel) && rotationLogger.Core().Enabled(zapcore.ErrorLevel) {
			cfg.Levels[2] = zapcore.ErrorLevel
		}
	}

	// 设置 PID 
	pid := utils.FormatInt(int64(os.Getpid()))

	var (
		once       sync.Once
		errHandler fiber.ErrorHandler
	)

	var errPadding = 15
	var latencyEnabled = contains("latency", cfg.Fields)

	skipURIs := make(map[string]struct{})
	for _, uri := range cfg.SkipURIs {
		skipURIs[uri] = struct{}{}
	}

	// 返回中间件的处理 Handler
	return func(c fiber.Ctx) (err error) {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		if _, ok := skipURIs[c.Path()]; ok {
			return c.Next()
		}

		once.Do(func() {
			stack := c.App().Stack()
			for m := range stack {
				for r := range stack[m] {
					if len(stack[m][r].Path) > errPadding {
						errPadding = len(stack[m][r].Path)
					}
				}
			}
			errHandler = c.App().Config().ErrorHandler
		})

		var start, stop time.Time

		if latencyEnabled {
			start = time.Now()
		}

		chainErr := c.Next()

		if chainErr != nil {
			if err := errHandler(c, chainErr); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		if latencyEnabled {
			stop = time.Now()
		}

		var (
			s     = c.Response().StatusCode()
			index int
		)
		switch {
		case s >= 500:
			index = 0
		case s >= 400:
			index = 1
		default:
			index = 2
		}
		levelIndex := index
		if levelIndex >= len(cfg.Levels) {
			levelIndex = len(cfg.Levels) - 1
		}
		messageIndex := index
		if messageIndex >= len(cfg.Messages) {
			messageIndex = len(cfg.Messages) - 1
		}

		ce := cfg.Logger.Check(cfg.Levels[levelIndex], cfg.Messages[messageIndex])
		if ce == nil {
			return nil
		}

		fields := make([]zap.Field, 0, len(cfg.Fields)+1)
		fields = append(fields, zap.Error(err))

		if cfg.FieldsFunc != nil {
			fields = append(fields, cfg.FieldsFunc(c)...)
		}

		for _, field := range cfg.Fields {
			switch field {
			case "referer":
				fields = append(fields, zap.String("referer", c.Get(fiber.HeaderReferer)))
			case "protocol":
				fields = append(fields, zap.String("protocol", c.Protocol()))
			case "pid":
				fields = append(fields, zap.String("pid", pid))
			case "port":
				fields = append(fields, zap.String("port", c.Port()))
			case "ip":
				fields = append(fields, zap.String("ip", c.IP()))
			case "ips":
				fields = append(fields, zap.String("ips", c.Get(fiber.HeaderXForwardedFor)))
			case "host":
				fields = append(fields, zap.String("host", c.Hostname()))
			case "path":
				fields = append(fields, zap.String("path", c.Path()))
			case "url":
				fields = append(fields, zap.String("url", c.OriginalURL()))
			case "ua":
				fields = append(fields, zap.String("ua", c.Get(fiber.HeaderUserAgent)))
			case "latency":
				fields = append(fields, zap.String("latency", stop.Sub(start).String()))
			case "status":
				fields = append(fields, zap.Int("status", c.Response().StatusCode()))
			case "resBody":
				if cfg.SkipResBody == nil || !cfg.SkipResBody(c) {
					if cfg.GetResBody == nil {
						fields = append(fields, zap.ByteString("resBody", c.Response().Body()))
					} else {
						fields = append(fields, zap.ByteString("resBody", cfg.GetResBody(c)))
					}
				}
			case "queryParams":
				fields = append(fields, zap.String("queryParams", c.Request().URI().QueryArgs().String()))
			case "body":
				if cfg.SkipBody == nil || !cfg.SkipBody(c) {
					fields = append(fields, zap.ByteString("body", c.Body()))
				}
			case "bytesReceived":
				fields = append(fields, zap.Int("bytesReceived", len(c.Request().Body())))
			case "bytesSent":
				fields = append(fields, zap.Int("bytesSent", len(c.Response().Body())))
			case "route":
				fields = append(fields, zap.String("route", c.Route().Path))
			case "method":
				fields = append(fields, zap.String("method", c.Method()))
			case "requestId":
				fields = append(fields, zap.String("requestId", c.GetRespHeader(fiber.HeaderXRequestID)))
			case "error":
				if chainErr != nil {
					fields = append(fields, zap.String("error", chainErr.Error()))
				}
			case "reqHeaders":
				for header, values := range c.GetReqHeaders() {
					if len(values) == 0 {
						continue
					}

					sanitized := sanitizeHeaderValues(header, values)

					if len(sanitized) == 1 {
						fields = append(fields, zap.String(header, sanitized[0]))
						continue
					}

					fields = append(fields, zap.Strings(header, sanitized))
				}
			}
		}

		ce.Write(fields...)

		return nil
	}
}

func contains(needle string, slice []string) bool {
	for _, e := range slice {
		if e == needle {
			return true
		}
	}
	return false
}

var sensitiveRequestHeaders = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"x-api-key":           {},
	"x-auth-token":        {},
}

func sanitizeHeaderValues(header string, values []string) []string {
	if len(values) == 0 {
		return values
	}

	if _, ok := sensitiveRequestHeaders[utilsstrings.ToLower(header)]; !ok {
		return values
	}

	sanitized := make([]string, len(values))
	for i := range sanitized {
		sanitized[i] = "[REDACTED]"
	}

	return sanitized
}