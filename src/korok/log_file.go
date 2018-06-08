package korok

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	CUT_CHECK_INTERVAL = 20
)

func NewLogFile(conf *LogConf) *LogFile {
	lf := &LogFile{
		FileName: conf.Name,
		FilePath: conf.FilePath,
		FileSize: conf.FileSize,
	}
	lf.Init()
	return lf
}

type LogFile struct {
	FileName string
	FilePath string
	FileSize int64

	SinceLastCheck int
	LastCheckTime  time.Time

	NormalFileName string
	FatalFileName  string

	FileMu sync.Mutex

	NormalFile *os.File
	FatalFile  *os.File
}

func (lf *LogFile) Init() {

	err := os.MkdirAll(lf.FilePath, 0755)
	if err != nil {
		panic(fmt.Sprintf("create log path failed: %s", err))
	}

	lf.NormalFileName = lf.FilePath + "/" + lf.FileName + ".log"
	file, err := lf.OpenOrCreateFile(lf.NormalFileName)
	if err != nil {
		panic(fmt.Sprintf("open NormalFile failed: %s", err))
	}
	lf.NormalFile = file

	lf.FatalFileName = lf.FilePath + "/" + lf.FileName + ".log.wf"
	file, err = lf.OpenOrCreateFile(lf.FatalFileName)
	if err != nil {
		panic(fmt.Sprintf("open FatalFile failed: %s", err))
	}
	lf.FatalFile = file

	lf.LastCheckTime = time.Now().Local()
}

func (lf *LogFile) CloseFile() {
	lf.NormalFile.Close()
	lf.FatalFile.Close()
}

func (lf *LogFile) NeedCut() bool {
	defer func() {
		lf.SinceLastCheck = (lf.SinceLastCheck + 1) % CUT_CHECK_INTERVAL
	}()

	if lf.SinceLastCheck != 0 {
		return false
	}

	fileStat, err := lf.NormalFile.Stat()
	if err != nil {
		return false
	}

	defer func() {
		lf.LastCheckTime = time.Now().Local()
	}()

	now := time.Now().Local()
	if now.Hour()-lf.LastCheckTime.Hour() != 0 {
		return true
	}

	if fileStat.Size() > lf.FileSize {
		return true
	}

	return false
}

func (lf *LogFile) AutoCut() {
	if !lf.NeedCut() {
		return
	}

	lf.FileMu.Lock()
	defer lf.FileMu.Unlock()

	lf.CloseFile()

	timeStamp := fmt.Sprintf("%d%02d%02d%02d",
		lf.LastCheckTime.Year(),
		int(lf.LastCheckTime.Month()),
		lf.LastCheckTime.Day(),
		lf.LastCheckTime.Hour())
	tryIndex := 0
	backupNormalFileName := fmt.Sprintf("%s.%s", lf.NormalFileName, timeStamp)
	for {
		_, err := os.Stat(backupNormalFileName)
		if os.IsNotExist(err) {
			break
		}
		tryIndex = tryIndex + 1
		backupNormalFileName = fmt.Sprintf("%s.%s_%d", lf.NormalFileName, timeStamp, tryIndex)
	}

	backupFatalFileName := fmt.Sprintf("%s.%s", lf.FatalFileName, timeStamp)
	if tryIndex != 0 {
		backupFatalFileName = fmt.Sprintf("%s_%d", backupFatalFileName, tryIndex)
	}

	os.Rename(lf.NormalFileName, backupNormalFileName)
	os.Rename(lf.FatalFileName, backupFatalFileName)

	lf.NormalFile, _ = lf.OpenOrCreateFile(lf.NormalFileName)
	lf.FatalFile, _ = lf.OpenOrCreateFile(lf.FatalFileName)

}

func (lf *LogFile) OpenOrCreateFile(fileName string) (*os.File, error) {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	return file, err
}

func (lf *LogFile) Flush(data []byte) (int, error) {

	lf.AutoCut()

	len, err := lf.NormalFile.Write(data)

	return len, err
}

func (lf *LogFile) FlushFatal(data []byte) (int, error) {
	lf.FileMu.Lock()
	defer lf.FileMu.Unlock()

	len, err := lf.FatalFile.Write(data)

	return len, err
}
