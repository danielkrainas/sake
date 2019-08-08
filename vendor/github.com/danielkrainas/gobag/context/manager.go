package bagcontext

import (
	"context"
	"net/http"
	"sync"
)

type Manager struct {
	contexts map[*http.Request]context.Context
	mutex    sync.Mutex
}

var DefaultContextManager = NewManager()

func NewManager() *Manager {
	return &Manager{
		contexts: make(map[*http.Request]context.Context),
	}
}

func (m *Manager) Context(parent context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if ctx, ok := m.contexts[r]; ok {
		return ctx
	}

	if parent == nil {
		parent = Background()
	}

	ctx := WithRequest(parent, r)
	ctx, _ = WithResponseWriter(ctx, w)
	ctx = WithLogger(ctx, GetLogger(ctx))
	m.contexts[r] = ctx
	return ctx
}

func (m *Manager) Release(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	r, err := GetRequest(ctx)
	if err != nil {
		GetLogger(ctx).Error("no request found in context at release")
		return
	}

	delete(m.contexts, r)
}
