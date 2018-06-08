package korok

import (
	"io"
	"sync"
	"time"
)

type AsyncWriter interface {
	io.Writer
	WriteFatal(p []byte)
	Stop()
}

func NewAsyncLogging(conf *LogConf) *AsyncLogging {
	alog := &AsyncLogging{
		LogFile:     NewLogFile(conf),
		PagePool:    NewBufPoolWithSize(conf.FlushThreshold),
		PageChannel: make(chan *Buffer, 100),
		stopChannel: make(chan int, 1),
	}
	alog.StartRoutine()
	return alog
}

type AsyncLogging struct {
	LogFile *LogFile

	isStop      bool
	stopChannel chan int
	wait        sync.WaitGroup

	mu sync.Mutex

	FreePage    *Buffer
	PagePool    *FixedSizeBufPool
	PageChannel chan *Buffer
}

func (alog *AsyncLogging) Write(logline []byte) (n int, err error) {

	if alog.isStop {
		return alog.LogFile.Flush(logline)
	}

	lineSize := len(logline)

	alog.mu.Lock()
	defer alog.mu.Unlock()

	if alog.FreePage == nil {
		alog.FreePage = alog.PagePool.Get()
	}

	if lineSize >= alog.FreePage.AvailSize() {
		alog.RenewCurrPageWithoutLock()
	}

	alog.FreePage.AppendByteSlice(logline)
	// FIXME
	return lineSize, nil
}

func (alog *AsyncLogging) WriteFatal(logline []byte) {
	alog.LogFile.FlushFatal(logline)
}

func (alog *AsyncLogging) RenewCurrPageWithoutLock() {
	if alog.FreePage == nil {
		return
	}

	alog.PageChannel <- alog.FreePage
	alog.FreePage = alog.PagePool.Get()
}

func (alog *AsyncLogging) WriteAndFreePage(page *Buffer) {
	alog.LogFile.Flush(page.Bytes())
	alog.PagePool.Free(page)
}

func (alog *AsyncLogging) SyncFlush() {
	defer alog.wait.Done()
	for {
		select {
		case fullPage := <-alog.PageChannel:
			alog.WriteAndFreePage(fullPage)
		default:
			alog.mu.Lock()
			duePage := alog.FreePage
			alog.FreePage = nil
			alog.mu.Unlock()

			if duePage != nil {
				alog.WriteAndFreePage(duePage)
			}

			return
		}
	}
}

func (alog *AsyncLogging) AsyncFlush() {

	alog.wait.Add(1)
	clocker := time.NewTicker(time.Duration(1000) * time.Millisecond)
	for {
		if alog.isStop == true {
			return
		}

		select {
		case <-alog.stopChannel:
			if alog.isStop == true {
				return
			}
		case <-clocker.C:

			alog.mu.Lock()
			alog.RenewCurrPageWithoutLock()
			alog.mu.Unlock()

		case fullPage := <-alog.PageChannel:
			alog.WriteAndFreePage(fullPage)
		}
	}
}

func (alog *AsyncLogging) StartRoutine() {
	go func() {
		defer alog.SyncFlush()
		alog.AsyncFlush()
	}()
}

func (alog *AsyncLogging) Stop() {
	alog.isStop = true
	alog.stopChannel <- 1
	alog.wait.Wait()
}
