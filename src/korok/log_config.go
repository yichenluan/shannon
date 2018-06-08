package korok

type LogConf struct {
	Level          int
	Name           string
	FilePath       string
	FileSize       int64
	FlushThreshold int32
	CallerSkip     int
}

func NewLogConf() *LogConf {
	return &LogConf{
		Level:          3,
		Name:           "shannon",
		FilePath:       "../log",
		FileSize:       2000000000,
		FlushThreshold: 4000 * 1000,
		CallerSkip:     3,
	}
}

func (conf *LogConf) SetLevel(level int) {
	conf.Level = level
}

func (conf *LogConf) SetName(name string) {
	conf.Name = name
}

func (conf *LogConf) SetFilePath(path string) {
	conf.FilePath = path
}

func (conf *LogConf) SetFileSize(size int64) {
	conf.FileSize = size
}

func (conf *LogConf) SetFlushThreshold(ts int32) {
	conf.FlushThreshold = ts
}

func (conf *LogConf) SetCallerSkip(skip int) {
	conf.CallerSkip = skip
}
