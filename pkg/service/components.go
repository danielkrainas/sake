package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/danielkrainas/sake/pkg/util/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Component interface {
	ComponentName() string
	Run(ctx ComponentRunContext) error
}

func ComponentField(c Component) zap.Field {
	return zap.String("component", c.ComponentName())
}

func runComponent(ctx ComponentRunContext, c Component) {
	log.Info("component started", ComponentField(c))
	defer func() {
		defer ctx.Done()
		log.Info("component stopped", ComponentField(c))
		if pobj := recover(); pobj != nil {
			log.Error("component failure", ComponentField(c), zap.Any("reason", pobj))
		}
	}()

	if err := c.Run(ctx); err != nil {
		log.Error("component error", zap.Error(err))
	}
}

type ComponentRunContext struct {
	QuitCh chan struct{}
	wg     *sync.WaitGroup
}

func (ctx ComponentRunContext) Done() {
	ctx.wg.Done()
}

type ComponentManager struct {
	contexts    map[Component]*ComponentRunContext
	activeFlag  int32
	wg          *sync.WaitGroup
	contextLock sync.Mutex
}

func NewComponentManager() *ComponentManager {
	cm := &ComponentManager{
		contexts:   make(map[Component]*ComponentRunContext),
		activeFlag: 0,
		wg:         &sync.WaitGroup{},
	}

	return cm
}

func (cm *ComponentManager) Active() bool {
	return atomic.LoadInt32(&cm.activeFlag) != 0
}

func (cm *ComponentManager) MustUse(c Component) {
	if cm.Active() {
		panic(errors.New("cannot modify running component manager"))
	}

	cm.contexts[c] = nil
}

func (cm *ComponentManager) Run() error {
	if cm.Active() {
		return errors.New("component manager already running")
	}

	defer atomic.StoreInt32(&cm.activeFlag, 0)
	defer log.Info("component manager stopped")
	atomic.StoreInt32(&cm.activeFlag, 1)
	log.Info("component manager started")
	wg := &sync.WaitGroup{}
	func() {
		cm.contextLock.Lock()
		defer cm.contextLock.Unlock()
		cm.wg = wg
		wg.Add(len(cm.contexts))
		for c := range cm.contexts {
			ctx := &ComponentRunContext{
				QuitCh: make(chan struct{}),
				wg:     wg,
			}

			cm.contexts[c] = ctx
			go runComponent(*ctx, c)
		}
	}()

	wg.Wait()
	return nil
}

func (cm *ComponentManager) Shutdown() {
	if cm.Active() {
		cm.contextLock.Lock()
		defer cm.contextLock.Unlock()
		for _, ctx := range cm.contexts {
			if ctx != nil {
				ctx.QuitCh <- struct{}{}
				close(ctx.QuitCh)
			}
		}

		cm.wg.Wait()
	}
}

type RunTasker interface {
	RunTask() error
}

type TaskComponent struct {
	name     string
	taskName string
	LogLevel zapcore.Level
	Interval time.Duration
	Tasker   RunTasker
}

var _ Component = (*TaskComponent)(nil)

func NewTaskComponent(name string, interval time.Duration, logLevel zapcore.Level, tasker RunTasker) *TaskComponent {
	return &TaskComponent{
		name:     "task_" + name,
		taskName: name,
		Interval: interval,
		Tasker:   tasker,
		LogLevel: logLevel,
	}
}

func (tc *TaskComponent) ComponentName() string {
	return tc.name
}

func (tc *TaskComponent) Run(ctx ComponentRunContext) error {
	time.Sleep(5 * time.Second) // allow time to warm up
	timer := time.NewTicker(tc.Interval)
	for {
		log.At(tc.LogLevel, "task execute", zap.String("task", tc.taskName))
		if err := tc.Tasker.RunTask(); err != nil {
			log.Error("task fail", zap.String("task", tc.taskName), zap.Error(err))
		} else {
			log.At(tc.LogLevel, "task success", zap.String("task", tc.taskName))
		}

		select {
		case <-ctx.QuitCh:
			return nil
		case <-timer.C:
		}
	}
}
