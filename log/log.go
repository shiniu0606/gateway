package log

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
    "time"
)

//var logger *zap.Logger
var errorLogger *zap.SugaredLogger

func InitLog(logPath string,errorPath string){

	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
        MessageKey:  "msg",
        LevelKey:    "level",
        EncodeLevel: zapcore.CapitalLevelEncoder,
        TimeKey:     "ts",
        EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
            enc.AppendString(t.Format("2006-01-02 15:04:05"))
        },
        CallerKey:    "file",
        EncodeCaller: zapcore.ShortCallerEncoder,
        EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
            enc.AppendInt64(int64(d) / 1000000)
        },
    })

	// 实现两个判断日志等级的interface (其实 zapcore.*Level 自身就是 interface)
    infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
        return lvl < zapcore.WarnLevel
    })
    
    warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
        return lvl >= zapcore.WarnLevel
    })

	// 获取 info、warn日志文件的io.Writer 抽象 getWriter() 在下方实现
    infoWriter := getWriter(logPath)
    warnWriter := getWriter(errorPath)
    
    // 最后创建具体的Logger
    core := zapcore.NewTee(
    	zapcore.NewCore(encoder, zapcore.AddSync(infoWriter), infoLevel),
        zapcore.NewCore(encoder, zapcore.AddSync(warnWriter), warnLevel),
    )
    
	log:= zap.New(core, zap.AddCaller())
	errorLogger = log.Sugar()
}

func getWriter(filename string) io.Writer {
    hook, err := rotatelogs.New(
        filename+".%Y%m%d%H",
        rotatelogs.WithLinkName(filename),
        rotatelogs.WithMaxAge(time.Hour*24*7),
        rotatelogs.WithRotationTime(time.Hour),
    )   

    if err != nil {
        panic(err)
    }   
    return hook
}

func Debug(args ...interface{}) {
    errorLogger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
    errorLogger.Debugf(template, args...)
}

func Info(args ...interface{}) {
    errorLogger.Info(args...)
}

func Infof(template string, args ...interface{}) {
    errorLogger.Infof(template, args...)
}

func Warn(args ...interface{}) {
    errorLogger.Warn(args...)
}
func Warnf(template string, args ...interface{}) {
    errorLogger.Warnf(template, args...)
}

func Error(args ...interface{}) {
    errorLogger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
    errorLogger.Errorf(template, args...)
}

func DPanic(args ...interface{}) {
    errorLogger.DPanic(args...)
}

func DPanicf(template string, args ...interface{}) {
    errorLogger.DPanicf(template, args...)
}

func Panic(args ...interface{}) {
    errorLogger.Panic(args...)
}

func Panicf(template string, args ...interface{}) {
    errorLogger.Panicf(template, args...)
}

func Fatal(args ...interface{}) {
    errorLogger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
    errorLogger.Fatalf(template, args...)
}