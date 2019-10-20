package trout

import (
	"context"
	"net/http"
	"strings"
	"sync"
)

type Router struct {
	node      *node
	finalized bool

	paramsPool sync.Pool

	NotFound         http.Handler
	MethodNotAllowed http.Handler
	PanicHandler     func(http.ResponseWriter, *http.Request, interface{})
}

type Param struct {
	Key   string
	Value string
}

type Params []*Param

func (ps Params) ByName(name string) string {
	for i := range ps {
		if ps[i].Key == name {
			return ps[i].Value
		}
	}
	return ""
}

type paramsKey struct{}

var ParamsKey = paramsKey{}

func ParamsFromContext(ctx context.Context) Params {
	p, _ := ctx.Value(ParamsKey).(Params)
	return p
}

func New() *Router {
	m := &Router{
		node: newNode(""),
	}
	m.paramsPool.New = func() interface{} {
		return &Params{}
	}
	return m
}

func (r *Router) GET(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodGet, h)
}

func (r *Router) HEAD(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodHead, h)
}

func (r *Router) OPTIONS(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodOptions, h)
}

func (r *Router) POST(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodPost, h)
}

func (r *Router) PUT(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodPut, h)
}

func (r *Router) PATCH(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodPatch, h)
}

func (r *Router) DELETE(path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(http.MethodDelete, h)
}

func (r *Router) Handle(method string, path string, h http.Handler) {
	r.node.getPathNode(path).addMethod(method, h)
}

func (r *Router) Lookup(method string, path string) (h http.Handler, ps Params, found bool) {
	node, pars, found := r.node.match(path, r.getParams)
	if found && node != nil && node.methods != nil {
		h = node.methods[strings.ToUpper(method)]
		if h != nil {
			ps = *pars
			found = true
		}
	}
	return
}

func (r *Router) recv(w http.ResponseWriter, req *http.Request) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(w, req, rcv)
	}
}

func (r *Router) getParams() *Params {
	ps := r.paramsPool.Get().(*Params)
	*ps = (*ps)[0:0]
	return ps
}

func (r *Router) putParams(ps *Params) {
	if ps != nil {
		r.paramsPool.Put(ps)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}

	if !r.finalized {
		r.node.finalize()
		r.finalized = true
	}

	node, ps, found := r.node.match(req.URL.Path, r.getParams)
	defer r.putParams(ps)
	if found && node != nil && node.methods != nil {
		m := node.methods[strings.ToUpper(req.Method)]
		if m != nil {
			if node.paramsCount > 0 {
				ctx := req.Context()
				ctx = context.WithValue(ctx, ParamsKey, *ps)
				req = req.WithContext(ctx)
			}
			m.ServeHTTP(w, req)
		} else {
			w.Header().Set("Allow", strings.Join(node.allowed, ", "))
			if r.MethodNotAllowed != nil {
				r.MethodNotAllowed.ServeHTTP(w, req)
			} else {
				http.Error(w,
					http.StatusText(http.StatusMethodNotAllowed),
					http.StatusMethodNotAllowed,
				)
			}
			return
		}

	} else {
		if r.NotFound != nil {
			r.NotFound.ServeHTTP(w, req)
		} else {
			http.NotFound(w, req)
		}
	}
}
