package model

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/ghodss/yaml"
	"jstio/internel/logs"
	"jstio/internel/util"
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

	clean := util.RemoveMultiBlankLineEx(buf.String())

	return []byte(clean), nil
}

func (rt *ResourceTemplateRender) RenderDelta(resType ResourceType, app *Application) (string, error) {
	t, ok := rt.deltaTemplates_[resType]
	if !ok {
		return "", errors.New("invalid resource type")
	}

	buf := bytes.Buffer{}
	bw := bufio.NewWriter(&buf)
	err := t.ExecuteTemplate(bw, resType, app)
	if err != nil {
		return "", err
	}
	_ = bw.Flush()

	clean := util.RemoveMultiBlankLine(buf.String())

	return clean, nil
}

// 更新app，更换上游时尝试将新添加的应用。愚蠢但有用。
func (rt *ResourceTemplateRender) TryMergeUpstreamResources(app *Application) error {
	var (
		err error
	)

	tagLog := logs.FuncTaggedLoggerFactory()

	for idx, res := range app.Resources {
		for _, upstream := range app.Upstream {
			upstreamHash := upstream.Hash()
			if !strings.Contains(res.YamlConfig, upstreamHash) {
				deltaRes, e := rt.RenderDelta(res.ResType, upstream)
				if e != nil {
					err = e
					tagLog(res.ResType).WithError(e).Errorln("app:", upstreamHash)
					continue
				}

				yamlConfig := util.RemoveMultiBlankLineEx(app.Resources[idx].YamlConfig +"\n" + deltaRes)
				jsonConfig, e := yaml.YAMLToJSON([]byte(yamlConfig))
				if err != nil {
					err = e
					tagLog(res.ResType).WithError(e).Errorf("app: %s, upstream: %s\n", app.Hash(), upstreamHash)
					continue
				}

				app.Resources[idx].YamlConfig = yamlConfig
				app.Resources[idx].Config = string(jsonConfig)

				if e = app.Resources[idx].Update(false); e != nil {
					tagLog(res.ResType).WithError(e).Errorf("app: %s, upstream: %s\n", app.Hash(), upstreamHash)
					continue
				}
				tagLog(res.ResType).Warningf("app: %s, upstream: %s, res:%s\n", app.Hash(), upstreamHash, deltaRes)
			}
		}
	}

	return err
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
