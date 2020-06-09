package model

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

var (
	renderOnce_ = sync.Once{}
	resRender   *ResourceTemplateRender
)

type ResourceTemplateRender struct {
	templates_      map[ResourceType]*template.Template
	deltaTemplates_ map[ResourceType]*template.Template
}

func newResourceTemplateRender() *ResourceTemplateRender {
	render := &ResourceTemplateRender{
		templates_:      make(map[ResourceType]*template.Template),
		deltaTemplates_: make(map[ResourceType]*template.Template),
	}

	render.loadTemplates()

	return render
}

func (rt *ResourceTemplateRender) loadTemplates() {
	var err error

	pwd, _ := os.Getwd()
	basePatten := path.Join(pwd, `static/xds_res`, "*.res")
	deltaPatten := path.Join(pwd, `static/xds_res`, "*.delta")

	basesGlob, _ := filepath.Glob(basePatten)
	deltasGlob, _ := filepath.Glob(deltaPatten)

	deltas := make([]string, 0)
	for _, delta := range deltasGlob {
		fb := filepath.Base(delta)
		fbx := strings.TrimSuffix(fb, filepath.Ext(fb))
		deltas = append(deltas, delta)
		rt.deltaTemplates_[fbx], err = template.New(fb).ParseFiles(delta)
		if err != nil {
			panic(err)
		}
	}

	for _, base := range basesGlob {
		fb := filepath.Base(base)
		fbx := strings.TrimSuffix(fb, filepath.Ext(fb))
		requires := append(deltas, base)
		rt.templates_[fbx], err = template.New(fb).ParseFiles(requires...)
		if err != nil {
			panic(err)
		}
	}
}

func (rt *ResourceTemplateRender) Render(resType ResourceType, app *Application) ([]byte, error) {

	t, ok := rt.templates_[resType]
	if !ok {
		return nil, errors.New("invalid resource type")
	}

	buf := bytes.Buffer{}
	bw := bufio.NewWriter(&buf)
	err := t.Execute(bw, app)
	if err != nil {
		return nil, err
	}
	_ = bw.Flush()

	return buf.Bytes(), nil
}

func (rt *ResourceTemplateRender) RenderDelta(resType ResourceType, data interface{}) ([]byte, error) {
	t, ok := rt.deltaTemplates_[resType]
	if !ok {
		return nil, errors.New("invalid resource type")
	}

	buf := bytes.Buffer{}
	bw := bufio.NewWriter(&buf)
	err := t.ExecuteTemplate(bw, resType, data)
	if err != nil {
		return nil, err
	}
	_ = bw.Flush()

	return buf.Bytes(), nil
}

func GetResourceRender() *ResourceTemplateRender {
	if resRender != nil {
		return resRender
	}

	renderOnce_.Do(func() {
		resRender = newResourceTemplateRender()
	})

	return resRender
}
