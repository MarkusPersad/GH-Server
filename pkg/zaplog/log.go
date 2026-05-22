package zaplog

import (
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logFile          = os.Getenv("LOG_FILE")
	logLevel         = os.Getenv("LOG_LEVEL")
	logFormat        = os.Getenv("LOG_TIME_FORMAT")
	logMaxSize, _    = strconv.ParseInt(os.Getenv("LOG_MAX_SIZE"), 10, 64)
	logMaxBackups, _ = strconv.ParseInt(os.Getenv("LOG_MAX_BACKUPS"), 10, 64)
	logMaxAge, _     = strconv.ParseInt(os.Getenv("LOG_MAX_AGE"), 10, 64)
	logCompress, _   = strconv.ParseBool(os.Getenv("LOG_COMPRESS"))
	Zap, _           = CustomLogger()
)

func CustomLogger() (*zap.Logger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "Time",
		LevelKey:       "Level",
		NameKey:        "Log",
		CallerKey:      "Caller",
		MessageKey:     "Message",
		StacktraceKey:  "StackTrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
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
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}
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
	zaplogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return zaplogger, nil
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(logFormat))
}
