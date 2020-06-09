package diskany

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultCacheSize = 8192
)

var (
	_instance *DiskLogServer
	_once     sync.Once
	_msgPool  = sync.Pool{
		New: func() interface{} {
			return &LogMessage{}
		},
	}
)

type LogMessage struct {
	path string
	buff bytes.Buffer
}

func GetLogMessage() *LogMessage {
	return _msgPool.Get().(*LogMessage)
}

func PutLogMessage(m *LogMessage) {
	m.buff.Reset()
	_msgPool.Put(m)
}

type LogFile struct {
	logSrv *DiskLogServer

	path       string
	autoRemove *time.Timer
	file       *os.File
}

type DiskLogServer struct {
	cache     chan *LogMessage
	logsTable map[string]*LogFile

	removeLog chan string

	stop chan struct{}

	locker sync.RWMutex
}

func (s *DiskLogServer) Remove() chan<- string {
	return s.removeLog
}

func (s *DiskLogServer) WriteBytes(pathName string, line []byte) error {
	msg := GetLogMessage()
	msg.path = pathName
	if _, err := msg.buff.Write(line); err != nil {
		return err
	}
	msg.buff.WriteByte('\n')

	select {

	case s.cache <- msg:
		return nil

	case <-time.After(time.Second):
		return errors.New("timeout: maybe channel has toooo many data")
	}
}

func (s *DiskLogServer) WriteString(pathName, line string) error {
	msg := GetLogMessage()
	msg.path = pathName
	if _, err := msg.buff.WriteString(line + "\n"); err != nil {
		return err
	}

	select {

	case s.cache <- msg:
		return nil

	case <-time.After(time.Second):
		return errors.New("timeout: maybe channel has toooo many data")
	}
}

func (s *DiskLogServer) logger(pathName string) *LogFile {
	s.locker.RLock()
	if o, ok := s.logsTable[pathName]; ok {
		s.locker.RUnlock()
		return o
	}
	s.locker.RUnlock()

	logger := &LogFile{
		path: pathName,
	}
	logger.make_(s)

	s.locker.Lock()
	if o, ok := s.logsTable[pathName]; ok {
		s.locker.Unlock()
		return o
	}
	s.logsTable[pathName] = logger

	s.locker.Unlock()

	return logger
}

func (s *DiskLogServer) autoServe() {
	for {
		select {
		case lm, ok := <-s.cache:
			if !ok {
				return
			}
			xlog := s.logger(lm.path)
			_, err := io.Copy(xlog.file, &lm.buff)
			if err != nil {
				fmt.Println("[logServer] io.Copy error:", err)
			}

			PutLogMessage(lm)
			xlog.Reset()

		case del, ok := <-s.removeLog:
			if !ok {
				return
			}
			s.locker.Lock()
			if o, ok := s.logsTable[del]; ok {
				o.file.Close()
				delete(s.logsTable, del)
			}
			s.locker.Unlock()

		case <-s.stop:
			return
		}
	}
}

func (s *DiskLogServer) Stop() {
	close(s.stop)

	// 尽量排空channel中的数据
	for {
		select {
		case lm, ok := <-s.cache:
			if !ok {
				return
			}
			xlog := s.logger(lm.path)
			_, err := io.Copy(xlog.file, &lm.buff)
			if err != nil {
				fmt.Println("[DiskLogServer] io.Copy error:", err)
			}
			PutLogMessage(lm)
			xlog.Reset()
		default:
			fmt.Println("[log-server] bye-bye!")
			return
		}
	}
}

func GetDiskLogger() *DiskLogServer {
	if _instance != nil {
		return _instance
	}

	_once.Do(func() {
		_instance = &DiskLogServer{
			locker:    sync.RWMutex{},
			cache:     make(chan *LogMessage, defaultCacheSize),
			stop:      make(chan struct{}),
			removeLog: make(chan string),
			logsTable: make(map[string]*LogFile),
		}

		go _instance.autoServe()
	})

	return _instance
}

func (lf *LogFile) make_(srv *DiskLogServer) {
	lf.autoRemove = time.AfterFunc(30*time.Second, func() {
		log.Println(lf.path, "will to close ...")
		lf.logSrv.Remove() <- lf.path
	})

	err := os.MkdirAll(filepath.Dir(lf.path), 0777)
	if err != nil {
		panic(err)
	}

	lf.file, err = os.OpenFile(lf.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		panic(err)
	}

	lf.logSrv = srv
}

func (lf *LogFile) Reset() {
	lf.autoRemove.Reset(30 * time.Second)
}
