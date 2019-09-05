package internel

import (
	"jstio/internel/logs"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

type exitHooksEntry struct {
	name string
	fn   func() error
}

type GracefulStopper struct {
	atExitHooks []exitHooksEntry
	locker      sync.Mutex
}

func NewGracefulStopper() *GracefulStopper {
	return &GracefulStopper{
		locker:      sync.Mutex{},
		atExitHooks: make([]exitHooksEntry, 0),
	}
}

func (gs *GracefulStopper) RegistryExitHook(name string, fn func() error) {
	gs.locker.Lock()
	defer gs.locker.Unlock()

	gs.atExitHooks = append(gs.atExitHooks, exitHooksEntry{name, fn})
}

func (gs *GracefulStopper) RunUntilStop(log *logrus.Logger) {
	sc := make(chan os.Signal)
	signal.Notify(sc)

	code := 0
	tagLog := logs.TaggedLoggerFactory(`atExitHooks`)

	for s := range sc {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP:
			for _, entry := range gs.atExitHooks {
				tagLog(entry.name).Println("executing signal:", s)
				e := entry.fn()
				if e != nil {
					code += 1 // FIXME: > 255
					tagLog(entry.name).Errorln(e)
				} else {
					tagLog(entry.name).Warningln("success")
				}
			}
			os.Exit(code)
		default:
			tagLog(s.String()).Println("un-expected signal caught")
		}
	}
}
