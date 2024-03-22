package views

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

type Template struct {
	htmlTpl *template.Template
	logger  *slog.Logger
}

func Must(t Template, err error) Template {
	if err != nil {
		panic(err)
	}
	return t
}

func ParseFS(logger *slog.Logger, fs fs.FS, pattern ...string) (Template, error) {
	htmlTpl, err := template.ParseFS(fs, pattern...)
	if err != nil {
		return Template{}, fmt.Errorf("parsing template: %w", err)
	}
	return Template{htmlTpl: htmlTpl, logger: logger}, nil
}

func (t Template) Execute(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := t.htmlTpl.Execute(w, data)
	if err != nil {
		t.logger.Error("Error occurred while executing template", "error", err.Error())
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
}
