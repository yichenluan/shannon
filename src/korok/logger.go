package korok

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	LOG_EVENT_NOTICE  = 0
	LOG_EVENT_ERROR   = 1
	LOG_EVENT_WARNING = 2
	LOG_EVENT_INFO    = 3
	LOG_EVENT_DEBUG   = 4
)

// LogIdLogger: Add LogId Info to Log Buf.
//
func NewLogIdLogger() *LogIdLogger {
	return &LogIdLogger{
		prefix: []byte("[logid "),
	}
}

type LogIdLogger struct {
	prefix []byte
}

func (ll *LogIdLogger) Handle(stream *Buffer, logid uint32) {
	stream.AppendByteSlice(ll.prefix)
	stream.AppendUint(uint64(logid))
	stream.AppendByte(']')
	stream.AppendByte(' ')

}

// CallerLogger: Add Caller Info to Log Buf.
//
func NewCallerLogger(depth int) *CallerLogger {
	return &CallerLogger{
		depth: depth,
	}
}

type CallerLogger struct {
	// TODO: Maybe A Pool
	depth int
}

func (cl *CallerLogger) Handle(stream *Buffer) {
	// TODO
	_, file, line, ok := runtime.Caller(cl.depth)
	if !ok {
		file = "???"
		line = 0
	}

	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			file = file[i+1:]
			break
		}
	}

	stream.AppendByte('[')
	stream.AppendString(file)
	stream.AppendByte(':')
	stream.AppendInt(int64(line))
	stream.AppendByte(']')
	stream.AppendByte(' ')

}

// SvrNameLogger: Add Server Name Info to Log Buf.
//
func NewSvrNameLogger(svrName string) *SvrNameLogger {
	return &SvrNameLogger{
		CachedSvrName: []byte(svrName),
	}
}

type SvrNameLogger struct {
	CachedSvrName []byte
}

func (sl *SvrNameLogger) Handle(stream *Buffer) {
	stream.AppendByteSlice(sl.CachedSvrName)
}

// TimeLogger: Add Time Info to Log Buf.
//
func NewTimeLogger() *TimeLogger {
	return &TimeLogger{}
}

type TimeLogger struct {
	mu sync.Mutex

	cachedStamp int64
	cachedTime  []byte
}

func (tl *TimeLogger) formatTime(now time.Time) {
	_, month, day := now.Date()
	hour, min, sec := now.Clock()
	formatStr := fmt.Sprintf("%02d-%02d %02d:%02d:%02d.", int(month), day, hour, min, sec)
	tl.cachedTime = []byte(formatStr)
}

func (tl *TimeLogger) initPerSecond() string {
	now := time.Now()
	currStamp := now.Unix()
	nano := fmt.Sprintf("%06d ", now.Nanosecond())

	if atomic.LoadInt64(&tl.cachedStamp) == currStamp {
		return nano
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()
	if tl.cachedStamp != currStamp {
		defer atomic.StoreInt64(&tl.cachedStamp, currStamp)
		tl.formatTime(now)
	}
	return nano
}

func (tl *TimeLogger) Handle(stream *Buffer) {
	nano := tl.initPerSecond()
	stream.AppendByteSlice(tl.cachedTime)
	stream.AppendString(nano)
}

// EventLogger: Add Event Info to Log Buf.
//
func NewEventLogger() *EventLogger {
	return &EventLogger{
		Notice:  []byte("NOTICE: "),
		Error:   []byte("FATAL: "),
		Warning: []byte("WARNING: "),
		Info:    []byte("INFO: "),
		Debug:   []byte("DEBUG: "),
	}
}

type EventLogger struct {
	Notice  []byte
	Error   []byte
	Warning []byte
	Info    []byte
	Debug   []byte
}

func (el *EventLogger) Handle(stream *Buffer, logEvent int) {
	if logEvent == LOG_EVENT_INFO {
		stream.AppendByteSlice(el.Info)
	} else if logEvent == LOG_EVENT_NOTICE {
		stream.AppendByteSlice(el.Notice)
	} else if logEvent == LOG_EVENT_ERROR {
		stream.AppendByteSlice(el.Error)
	} else if logEvent == LOG_EVENT_WARNING {
		stream.AppendByteSlice(el.Warning)
	} else if logEvent == LOG_EVENT_DEBUG {
		stream.AppendByteSlice(el.Debug)
	}
}

// Logger: Log FrontEnd.
//
// os.Stderr
func NewSuger() *Logger {
	//return New("Suger", os.Stderr, 3)
	log := &Logger{
		out:           os.Stderr,
		callerSkip:    3,
		svrName:       "Suger",
		level:         3,
		LogStreamPool: NewBufPoolWithSize(100),
	}
	log.InitHandler()
	return log
}

func NewMario() *Logger {
	conf := NewLogConf()
	return NewLogger(conf)
}

func NewLogger(conf *LogConf) *Logger {
	log := &Logger{
		out:           NewAsyncLogging(conf),
		callerSkip:    conf.CallerSkip,
		svrName:       conf.Name,
		level:         conf.Level,
		LogStreamPool: NewBufPoolWithSize(100),
	}
	log.InitHandler()
	return log
}

type Logger struct {
	eventEntry   *EventLogger
	timeEntry    *TimeLogger
	svrNameEntry *SvrNameLogger
	callerEntry  *CallerLogger
	logidEntry   *LogIdLogger

	out        io.Writer
	callerSkip int
	svrName    string
	level      int

	mu sync.Mutex

	LogStreamPool *FixedSizeBufPool
}

func (l *Logger) InitHandler() {
	//TODO
	l.eventEntry = NewEventLogger()
	l.timeEntry = NewTimeLogger()
	l.svrNameEntry = NewSvrNameLogger(l.svrName)
	l.callerEntry = NewCallerLogger(l.callerSkip)
	l.logidEntry = NewLogIdLogger()
}

func (l *Logger) complete(logid uint32, logEvent int, context string) *Buffer {
	stream := l.LogStreamPool.Get()

	l.eventEntry.Handle(stream, logEvent)
	l.timeEntry.Handle(stream)
	l.svrNameEntry.Handle(stream)
	l.callerEntry.Handle(stream)
	//l.logidEntry.Handle(stream, logid)

	stream.AppendString(context)
	stream.AppendByte('\n')

	return stream
}

func (l *Logger) LogWithEvent(event int, logid uint32, content string) {
	logBuffer := l.complete(logid, event, content)
	l.out.Write(logBuffer.Bytes())
	l.LogStreamPool.Free(logBuffer)
}

func (l *Logger) Debug(logid uint32, msg string, v ...interface{}) {
	if l.level < LOG_EVENT_DEBUG {
		return
	}

	l.LogWithEvent(LOG_EVENT_DEBUG, logid, fmt.Sprintf(msg, v...))
}

func (l *Logger) Notice(logid uint32, msg string, v ...interface{}) {
	if l.level < LOG_EVENT_NOTICE {
		return
	}

	l.LogWithEvent(LOG_EVENT_NOTICE, logid, fmt.Sprintf(msg, v...))
}

func (l *Logger) Info(logid uint32, msg string, v ...interface{}) {
	if l.level < LOG_EVENT_INFO {
		return
	}

	l.LogWithEvent(LOG_EVENT_INFO, logid, fmt.Sprintf(msg, v...))
}

func (l *Logger) Warn(logid uint32, msg string, v ...interface{}) {
	if l.level < LOG_EVENT_WARNING {
		return
	}

	l.LogWithEvent(LOG_EVENT_WARNING, logid, fmt.Sprintf(msg, v...))
}

func (l *Logger) Fatal(logid uint32, msg string, v ...interface{}) {
	if l.level < LOG_EVENT_ERROR {
		return
	}

	logBuffer := l.complete(logid, LOG_EVENT_ERROR, fmt.Sprintf(msg, v...))
	l.out.Write(logBuffer.Bytes())
	if alog, ok := l.out.(AsyncWriter); ok {
		alog.WriteFatal(logBuffer.Bytes())
	}
	l.LogStreamPool.Free(logBuffer)
}

func (l *Logger) Stop() {
	if alog, ok := l.out.(AsyncWriter); ok {
		alog.Stop()
	}
}
