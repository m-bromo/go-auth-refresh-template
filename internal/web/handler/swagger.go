package handler

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"

	"github.com/m-bromo/go-auth-template/docs"
)

//go:embed templates/swagger.html
var swaggerTemplateFS embed.FS

type swaggerTemplateData struct {
	SpecURL string
}

var swaggerTemplate = template.Must(template.ParseFS(swaggerTemplateFS, "templates/swagger.html"))

func RedirectSwagger(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
}

func SwaggerUI(w http.ResponseWriter, r *http.Request) {
	var body bytes.Buffer
	err := swaggerTemplate.Execute(&body, swaggerTemplateData{
		SpecURL: "/swagger/openapi.yaml",
	})
	if err != nil {
		http.Error(w, "failed to render swagger UI", http.StatusInternalServerError)
		return
	}

	HandleHTML(w, http.StatusOK, body)
}

func SwaggerSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(docs.OpenAPISpec); err != nil {
		return
	}
}
