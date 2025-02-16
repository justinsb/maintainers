package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/maintainers/experiments/keptain/pkg/model"
	"sigs.k8s.io/yaml"
)

// Repository represents a KEP repository
type Repository struct {
	basePath string
	keps     map[string]*model.KEP
}

// NewRepository creates a new KEP repository instance
func NewRepository(basePath string) (*Repository, error) {
	r := &Repository{
		basePath: basePath,
		keps:     make(map[string]*model.KEP),
	}
	if err := r.loadKEPs(); err != nil {
		return nil, fmt.Errorf("error loading KEPs: %v", err)
	}
	return r, nil
}

func (r *Repository) loadKEPs() error {
	// Walk the KEPs directory and load all KEPs
	if err := filepath.Walk(r.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// We assume there's a metadata file for each KEP called kep.yaml
		if filepath.Base(path) != "kep.yaml" {
			return nil
		}

		relativePath, err := filepath.Rel(r.basePath, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}

		dir := filepath.Dir(path)
		relativeDir := filepath.Dir(relativePath)

		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading KEP file: %w", err)
		}
		kep, err := r.parseKEPFile(b)
		if err != nil {
			// Log error but continue processing other KEPs
			return fmt.Errorf("error parsing KEP %q: %w", path, err)
		}

		// use the (repo-relative) directory as the identifier for the KEP
		kep.Path = relativeDir

		// See if we have a README.md file
		{
			readme := filepath.Join(dir, "README.md")
			readmeBytes, err := os.ReadFile(readme)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("error getting README.md: %w", err)
				}
				return nil
			}

			if err == nil {
				kep.TextContents = string(readmeBytes)
				kep.TextURL = fmt.Sprintf("https://github.com/kubernetes/enhancements/blob/master/%s", filepath.Join(relativeDir, "README.md"))
			}
		}
		r.keps[kep.Path] = kep
		return nil
	}); err != nil {
		return fmt.Errorf("error walking KEPs: %w", err)
	}

	return nil
}

// ListKEPs returns all KEPs in the repository
// If query is provided, it will filter the KEPs based on the query
func (r *Repository) ListKEPs(query string) ([]*model.KEP, error) {
	var ret []*model.KEP
	for _, kep := range r.keps {
		// Filter KEPs if search query is provided
		match := true
		if query != "" {
			query = strings.ToLower(query)
			if strings.Contains(strings.ToLower(kep.Title), query) ||
				strings.Contains(strings.ToLower(kep.Number), query) ||
				containsAuthor(kep.Authors, query) {
				match = true
			}
		}

		if match {
			ret = append(ret, kep)
		}
	}
	return ret, nil
}

func containsAuthor(authors []string, query string) bool {
	for _, author := range authors {
		if strings.Contains(strings.ToLower(author), query) {
			return true
		}
	}
	return false
}

// GetKEP returns a specific KEP by number
func (r *Repository) GetKEP(path string) (*model.KEP, error) {
	kep, ok := r.keps[path]
	if ok {
		return kep, nil
	}
	return nil, fmt.Errorf("KEP %s not found", path)
}

// kepFile is the format used in the KEP file.
type kepFile struct {
	Title             string   `json:"title"`
	Number            string   `json:"kep-number"`
	Authors           []string `json:"authors"`
	OwningSig         string   `json:"owning-sig"`
	ParticipatingSigs []string `json:"participating-sigs"`
	Reviewers         []string `json:"reviewers"`
	Approvers         []string `json:"approvers"`
	Editor            string   `json:"editor"`
	CreationDate      string   `json:"creation-date"`
	LastUpdated       string   `json:"last-updated"`
	Status            string   `json:"status"`
	SeeAlso           []string `json:"see-also"`
	Replaces          []string `json:"replaces"`
	SupersededBy      []string `json:"superseded-by"`
}

// parseKEPFile parses a KEP yaml file
func (r *Repository) parseKEPFile(data []byte) (*model.KEP, error) {

	var kep kepFile
	if err := yaml.Unmarshal(data, &kep); err != nil {
		return nil, fmt.Errorf("error parsing KEP yaml: %v", err)
	}

	// Extract additional metadata from the yaml
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("error parsing KEP metadata: %v", err)
	}

	out := &model.KEP{
		Title:   kep.Title,
		Number:  kep.Number,
		Authors: kep.Authors,
		Status:  kep.Status,
	}
	return out, nil
}
