package handlers

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/mtzanidakis/budgeting/internal/version"
)

// StaticHandler handles serving of template-based static files with version injection
type StaticHandler struct {
	staticFS      embed.FS
	indexTemplate *template.Template
	swTemplate    *template.Template
}

// TemplateData holds data for template rendering
type TemplateData struct {
	Version string
}

// NewStaticHandler creates a new static handler with parsed templates
func NewStaticHandler(staticFS embed.FS) (*StaticHandler, error) {
	// Parse index.html template
	indexTmpl, err := template.ParseFS(staticFS, "frontend/index.html.tmpl")
	if err != nil {
		return nil, err
	}

	// Parse service worker template
	swTmpl, err := template.ParseFS(staticFS, "frontend/sw.js.tmpl")
	if err != nil {
		return nil, err
	}

	return &StaticHandler{
		staticFS:      staticFS,
		indexTemplate: indexTmpl,
		swTemplate:    swTmpl,
	}, nil
}

// ServeIndexHTML renders and serves the index.html template
func (h *StaticHandler) ServeIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	data := TemplateData{
		Version: version.Get(),
	}

	if err := h.indexTemplate.Execute(w, data); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// ServeServiceWorker renders and serves the service worker template
func (h *StaticHandler) ServeServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	data := TemplateData{
		Version: version.Get(),
	}

	if err := h.swTemplate.Execute(w, data); err != nil {
		http.Error(w, "Failed to render service worker", http.StatusInternalServerError)
		return
	}
}
