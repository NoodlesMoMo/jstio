package service

import (
	"git.sogou-inc.com/iweb/jstio/internel"
	. "git.sogou-inc.com/iweb/jstio/internel/logs"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/qiangxue/fasthttp-routing"
)

const (
	templateRootDir = `static/templates`
)

var (
	templates_ map[string]*template.Template
)

func LoadDashboardTemplates() {
	templateReload()

	if internel.GetAfxOption().DebugMode {
		go templateAutoReload()
	}
}

func templateReload() {
	if templates_ == nil {
		templates_ = make(map[string]*template.Template)
	}

	bases, err := filepath.Glob(TemplateDir(templateRootDir, "base/*.html"))
	if err != nil {
		panic(err)
	}

	includes, err := filepath.Glob(TemplateDir(templateRootDir, "include/*.html"))

	baseParts := make(map[string][]string)
	for i := 0; i < len(bases); i++ {
		base := bases[i]
		seps := strings.SplitN(filepath.Base(base), "__", 2)
		if len(seps) > 1 {
			xb := seps[0]
			baseParts[xb] = append(baseParts[xb], base)
			bases = append(bases[:i], bases[i+1:]...)
			i--
		}
	}

	for _, base := range bases {
		files := append(includes, base)
		fb := filepath.Base(base)
		fbx := strings.TrimRight(fb, filepath.Ext(fb))
		if parts, ok := baseParts[fbx]; ok {
			files = append(files, parts...)
		}

		templates_[fb] = template.Must(template.ParseFiles(files...))
	}
}

func templateAutoReload() {
	logger := Logger
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithField("template", "auto_reload").Errorln(err)
		return
	}
	defer watcher.Close()

	done := make(chan struct{})

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if (event.Op&fsnotify.Rename == fsnotify.Rename) && (event.Op&fsnotify.Chmod == fsnotify.Chmod) {
					logger.WithField(`template`, `event`).Println("template will reload")
					templateReload()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				logger.WithField(`template`, `error`).Errorln(err)
			}
		}
	}()

	err = watcher.Add(TemplateDir(templateRootDir, "base"))
	if err != nil {
		logger.WithField(`template`, `error`).Errorln(err)
	}

	<-done
}

func TemplateDir(sep ...string) string {
	pwd, _ := os.Getwd()
	pwds := []string{pwd}
	for _, s := range sep {
		pwds = append(pwds, s)
	}

	return path.Join(pwds...)
}

func RenderTemplate(ctx *routing.Context, name string, data interface{}) error {
	ctx.SetContentType(`text/html; charset=utf-8`)

	bw := ctx.Response.BodyWriter()
	templ, ok := templates_[name]
	if !ok {
		return templates_[`404.html`].ExecuteTemplate(bw, `404.html`, nil)
	}

	return templ.ExecuteTemplate(bw, name, data)
}
