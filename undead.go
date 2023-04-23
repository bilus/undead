package undead

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	g "github.com/maragudk/gomponents"
)

type (
	Context struct {
		ginContext *gin.Context
		Event      string
		Params     Params
	}
	ModelConstructor[M any] func() *M
	Params                  struct {
		ginContext *gin.Context
	}
	View[M any]    func(m *M) g.Node
	Handler[M any] func(c *Context, m *M) error
	App[M any]     struct {
		newModel      ModelConstructor[M]
		paramHandlers []Handler[M]
		eventHandlers map[string]Handler[M]
	}
)

// TODO(bilus): Set model.LastError

func NewApp[M any](mc ModelConstructor[M]) *App[M] {
	gob.Register(*mc())

	return &App[M]{
		newModel:      mc,
		eventHandlers: (make(map[string]Handler[M])),
	}
}

func Middleware() gin.HandlerFunc {
	store := memstore.NewStore([]byte("secret"))
	return sessions.Sessions("acme-demo", store)
}

func (r *App[M]) HandleParams(h Handler[M]) {
	r.paramHandlers = append(r.paramHandlers, h)
}

func (r *App[M]) HandleEvent(eventID string, h Handler[M]) {
	r.eventHandlers[eventID] = h
}

func (r *App[M]) dispatch(c *Context) (*M, error) {
	m := r.loadModel(c)
	defer func() {
		err := r.saveModel(c, m)
		if err != nil {
			log.Printf("Error saving model: %v", err)
		}
	}()

	if err := do(r.paramHandlers, c, m); err != nil {
		return m, fmt.Errorf("error handling params: %w", err)
	}
	eventHandler, ok := r.eventHandlers[c.Event]
	if ok {
		if err := eventHandler(c, m); err != nil {
			log.Printf("Error handling event %s: %v", c.Event, err)
			return m, err
		}
	}
	return m, nil
}

func do[M any](handlers []Handler[M], c *Context, m *M) error {
	for i, h := range handlers {
		if err := h(c, m); err != nil {
			return fmt.Errorf("handler[M] #%d returned error: %w", i, err)
		}
	}
	return nil
}

func (r *App[M]) Handler(v View[M]) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := Params{c}
		context := Context{
			ginContext: c,
			Event:      params.String("event"),
			Params:     params,
		}
		m, err := r.dispatch(&context)
		if err != nil {
			log.Printf("Error during dispatch: %v", err)
		}
		buf := bytes.Buffer{}
		err = v(m).Render(&buf)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("%v", err),
			})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
	}
}

func (r *App[M]) loadModel(c *Context) *M {
	s := sessions.Default(c.ginContext)
	v := s.Get("model")
	if v != nil {
		m, ok := v.(M)
		if ok {
			return &m
		}
	}
	return r.newModel()
}

func (r *App[M]) saveModel(c *Context, m *M) error {
	s := sessions.Default(c.ginContext)
	s.Set("model", *m)
	return s.Save()
}

func (p Params) Int(k string) int {
	s := p.ginContext.PostForm(k)
	if s == "" {
		s = p.ginContext.Query(k)
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func (p Params) String(k string) string {
	s := p.ginContext.PostForm(k)
	if s == "" {
		s = p.ginContext.Query(k)
	}
	return s
}
