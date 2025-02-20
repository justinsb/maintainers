package website

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"k8s.io/klog/v2"
	"sigs.k8s.io/maintainers/experiments/keptain/pkg/model"
	"sigs.k8s.io/maintainers/experiments/keptain/pkg/store"
)

// Server is the main HTTP server for the website
type Server struct {
	kepRepo *store.Repository
}

// NewServer creates a new Server
func NewServer(kepRepo *store.Repository) *Server {
	return &Server{kepRepo: kepRepo}
}

// Run starts the server, and listens on the given endpoint forever.
func (s *Server) Run(endpoint string) error {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Routes
	mux.HandleFunc("GET /", s.handleHome)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /kep/{path...}", s.handleKEP)
	mux.HandleFunc("GET /pullrequests/", s.handlePullRequestList)

	fmt.Println("Server starting on :8080...")
	return http.ListenAndServe(endpoint, mux)
}

// HomePageData is the data model for the home page
type HomePageData struct {
	AllWorkflows []*model.KEP
	Query        string
}

// handleHome handles the home page, which is the list of all KEPs
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := klog.FromContext(ctx)

	query := r.URL.Query().Get("q")
	log.Info("Listing all workflows", "query", query)

	keps, err := s.kepRepo.ListKEPs(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading KEPs: %v", err), http.StatusInternalServerError)
		return
	}

	data := HomePageData{
		AllWorkflows: keps,
		Query:        query,
	}

	tmpl := template.Must(template.ParseFiles("templates/home.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}

// handleSearch handles the search fragment on the homepage, used when typing into the list.
// We may be able to harmonize this with handleHome in future.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := klog.FromContext(ctx)

	query := r.URL.Query().Get("q")
	log.Info("Searching KEPs", "query", query)

	keps, err := s.kepRepo.ListKEPs(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading KEPs: %v", err), http.StatusInternalServerError)
		return
	}

	data := HomePageData{
		AllWorkflows: keps,
		Query:        query,
	}

	tmpl := template.Must(template.ParseFiles("templates/home.html"))
	if err := tmpl.ExecuteTemplate(w, "kep_list", data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}

// KEPPageData is the data model for the KEP "detail" page
type KEPPageData struct {
	Workflow    *model.KEP
	ContentHTML template.HTML
}

// handleKEP handles the KEP "detail" page, which is the page that displays the KEP content.
func (s *Server) handleKEP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := klog.FromContext(ctx)

	path := r.PathValue("path")

	log.Info("Listing KEP", "path", path)

	kep, err := s.kepRepo.GetKEP(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading KEP: %v", err), http.StatusNotFound)
		return
	}

	// Idea: Maybe we should pre-render the markdown to HTML in the store,
	// and just serve the HTML here?

	// Configure markdown processor with GitHub Flavored Markdown
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
			extension.Table,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Required for GFM tables and task lists
			html.WithXHTML(),
		),
	)

	// Create a new parser context with TOC enabled
	context := parser.NewContext()

	// Parse the markdown content
	var buf bytes.Buffer
	doc := md.Parser().Parse(text.NewReader([]byte(kep.TextContents)), parser.WithContext(context))
	if err := md.Renderer().Render(&buf, []byte(kep.TextContents), doc); err != nil {
		http.Error(w, fmt.Sprintf("Error converting markdown: %v", err), http.StatusInternalServerError)
		return
	}

	data := KEPPageData{
		Workflow:    kep,
		ContentHTML: template.HTML(buf.String()),
	}

	tmpl := template.Must(template.ParseFiles("templates/kep.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}

// PullRequestsListPageData is the data model for the PR "list" page
type PullRequestsListPageData struct {
	PullRequests []*model.PullRequest
}

// handlePullRequestList handles the PR "list" page, which is the page that displays the list of pull requests against KEPs.
func (s *Server) handlePullRequestList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := klog.FromContext(ctx)

	log.Info("Listing KEP pull requests")

	prs, err := s.kepRepo.ListPullRequests()
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing pull requests: %v", err), http.StatusInternalServerError)
		return
	}

	data := PullRequestsListPageData{
		PullRequests: prs,
	}

	tmpl := template.Must(template.ParseFiles("templates/pullrequests/list.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}
