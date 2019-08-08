package bagcontext

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/danielkrainas/gobag/util/uid"
)

var (
	ErrNoRequestContext        = errors.New("no http request in context")
	ErrNoResponseWriterContext = errors.New("no http response in context")
)

type httpRequestContext struct {
	context.Context
	startedAt time.Time
	id        string
	r         *http.Request
}

func parseIP(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		log.Warnf("invalid remote IP address: %q", s)
	}

	return ip
}

func RemoteAddr(r *http.Request) string {
	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		proxies := strings.Split(prior, ",")
		if len(proxies) > 0 {
			remoteAddr := strings.Trim(proxies[0], " ")
			if parseIP(remoteAddr) != nil {
				return remoteAddr
			}
		}
	}

	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		if parseIP(realIP) != nil {
			return realIP
		}
	}

	return r.RemoteAddr
}

func RemoteIP(r *http.Request) string {
	addr := RemoteAddr(r)
	if ip, _, err := net.SplitHostPort(addr); err == nil {
		return ip
	}

	return addr
}

func WithRequest(ctx context.Context, r *http.Request) context.Context {
	if ctx.Value("http.request") != nil {
		panic("only one request per context")
	}

	return &httpRequestContext{
		Context:   ctx,
		startedAt: time.Now(),
		id:        uid.Generate(),
		r:         r,
	}
}

func GetRequest(ctx context.Context) (*http.Request, error) {
	if r, ok := ctx.Value("http.request").(*http.Request); r != nil && ok {
		return r, nil
	}

	return nil, ErrNoRequestContext
}

func GetRequestID(ctx context.Context) string {
	return GetStringValue(ctx, "http.request.id")
}

func (ctx *httpRequestContext) Value(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok {
		if keyStr == "http.request" {
			return ctx.r
		}

		for {
			if !strings.HasPrefix(keyStr, "http.request.") {
				break
			}

			parts := strings.Split(keyStr, ".")
			if len(parts) != 3 {
				break
			}

			switch parts[2] {
			case "query":
				return ctx.r.URL.Query()

			case "uri":
				return ctx.r.RequestURI

			case "remoteaddr":
				return RemoteAddr(ctx.r)

			case "method":
				return ctx.r.Method

			case "host":
				return ctx.r.Host

			case "referer":
				referer := ctx.r.Referer()
				if referer != "" {
					return referer
				}

			case "useragent":
				return ctx.r.UserAgent()

			case "id":
				return ctx.id

			case "startedat":
				return ctx.startedAt

			case "contenttype":
				contentType := ctx.r.Header.Get("Content-Type")
				if contentType != "" {
					return contentType
				}
			}

			break
		}
	}

	return ctx.Context.Value(key)
}

type muxVarsContext struct {
	context.Context
	vars map[string]string
}

func WithVars(ctx context.Context, r *http.Request) context.Context {
	return &muxVarsContext{
		Context: ctx,
		vars:    mux.Vars(r),
	}
}

func (ctx *muxVarsContext) Value(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok {
		if keyStr == "vars" {
			return ctx.vars
		}

		if strings.HasPrefix(keyStr, "vars.") {
			keyStr = strings.TrimPrefix(keyStr, "vars.")
		}

		if value, ok := ctx.vars[keyStr]; ok {
			return value
		}
	}

	return ctx.Context.Value(key)
}

func GetRequestQueryParameters(ctx context.Context) url.Values {
	if qv, ok := ctx.Value("http.request.query").(url.Values); ok {
		return qv
	}
	return url.Values{}
}

func GetRequestLogger(ctx context.Context) Logger {
	return GetLogger(ctx,
		"http.request.id",
		"http.request.method",
		"http.request.host",
		"http.request.uri",
		"http.request.referer",
		"http.request.useragent",
		"http.request.remoteaddr",
		"http.request.contenttype")
}

func GetResponseLogger(ctx context.Context) Logger {
	l := getLogrusLogger(ctx,
		"http.response.written",
		"http.response.status",
		"http.response.contenttype")

	duration := Since(ctx, "http.request.startedat")
	if duration > 0 {
		l = l.WithField("http.response.duration", duration.String())
	}

	return l
}

func GetResponseWriter(ctx context.Context) (http.ResponseWriter, error) {
	v := ctx.Value("http.response")

	w, ok := v.(http.ResponseWriter)
	if !ok || w == nil {
		return nil, ErrNoResponseWriterContext
	}

	return w, nil
}

func WithResponseWriter(ctx context.Context, w http.ResponseWriter) (context.Context, http.ResponseWriter) {
	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		iwcn := &instrumentedResponseWriterCloseNotify{
			instrumentedResponseWriter: instrumentedResponseWriter{
				ResponseWriter: w,
				Context:        ctx,
			},
			CloseNotifier: closeNotifier,
		}

		return iwcn, iwcn
	}

	iw := instrumentedResponseWriter{
		ResponseWriter: w,
		Context:        ctx,
	}
	return &iw, &iw
}

type instrumentedResponseWriterCloseNotify struct {
	instrumentedResponseWriter
	http.CloseNotifier
}

type instrumentedResponseWriter struct {
	http.ResponseWriter
	context.Context

	mutex   sync.Mutex
	status  int
	written int64
}

func (iw *instrumentedResponseWriter) Write(p []byte) (int, error) {
	n, err := iw.ResponseWriter.Write(p)

	iw.mutex.Lock()
	defer iw.mutex.Unlock()

	iw.written += int64(n)

	if iw.status == 0 {
		iw.status = http.StatusOK
	}

	return n, err
}

func (iw *instrumentedResponseWriter) WriteHeader(status int) {
	iw.ResponseWriter.WriteHeader(status)
	iw.mutex.Lock()
	iw.status = status
	iw.mutex.Unlock()
}

func (iw *instrumentedResponseWriter) Flush() {
	if flusher, ok := iw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (iw *instrumentedResponseWriter) Value(key interface{}) interface{} {
	if ks, ok := key.(string); ok {
		if ks == "http.response" {
			return iw
		}

		for {
			if !strings.HasPrefix(ks, "http.response.") {
				break
			}

			parts := strings.Split(ks, ".")
			if len(parts) != 3 {
				break
			}

			iw.mutex.Lock()
			defer iw.mutex.Unlock()

			switch parts[2] {
			case "written":
				return iw.written

			case "status":
				return iw.status

			case "contenttype":
				contentType := iw.Header().Get("Content-Type")
				if contentType != "" {
					return contentType
				}
			}

			break
		}
	}

	return iw.Context.Value(key)
}

func (iw *instrumentedResponseWriterCloseNotify) Value(key interface{}) interface{} {
	if ks, ok := key.(string); ok {
		if ks == "http.response" {
			return iw
		}
	}

	return iw.instrumentedResponseWriter.Value(key)
}
