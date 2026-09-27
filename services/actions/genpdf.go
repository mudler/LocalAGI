package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/jung-kurt/gofpdf"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	MetadataPDFs = "pdf_paths"
)

// NewGenPDF creates a new PDF generation action
func NewGenPDF(config map[string]string) *GenPDFAction {
	a := &GenPDFAction{
		outputDir:    config["outputDir"],
		cleanOnStart: config["cleanOnStart"] == "true" || config["cleanOnStart"] == "1",
	}

	if a.outputDir != "" {
		if err := os.MkdirAll(a.outputDir, 0755); err != nil {
			xlog.Error("Failed to create output directory", "path", a.outputDir, "error", err)
		}
		if a.cleanOnStart {
			entries, err := os.ReadDir(a.outputDir)
			if err == nil {
				for _, e := range entries {
					_ = os.Remove(filepath.Join(a.outputDir, e.Name()))
				}
			}
		}
	}

	return a
}

type GenPDFAction struct {
	outputDir    string
	cleanOnStart bool
}

func (a *GenPDFAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Filename string `json:"filename"`
	}{}
	if err := params.Unmarshal(&result); err != nil {
		return types.ActionResult{}, err
	}

	if result.Content == "" {
		return types.ActionResult{}, fmt.Errorf("content is required")
	}

	if a.outputDir == "" {
		return types.ActionResult{}, fmt.Errorf("outputDir is required for generate_pdf (configure the action with an output directory)")
	}

	// Generate filename if not provided
	filename := result.Filename
	if filename == "" {
		filename = fmt.Sprintf("document_%d", time.Now().UnixNano())
	}

	// Clean filename to prevent path traversal
	filename = filepath.Base(filename)

	// Ensure filename has .pdf extension
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		filename = filename + ".pdf"
	}

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Add title if provided
	if result.Title != "" {
		pdf.SetFont("Arial", "B", 16)
		pdf.MultiCell(0, 10, tr(result.Title), "", "", false)
		pdf.Ln(5)
	}

	// Add content: parse as markdown and render, or fall back to plain text
	pdf.SetFont("Arial", "", 12)
	p := parser.NewWithExtensions(parser.CommonExtensions)
	doc := p.Parse([]byte(result.Content))
	if doc != nil && ast.GetFirstChild(doc) != nil {
		renderMarkdownToPDF(pdf, tr, doc)
	} else {
		pdf.MultiCell(0, 10, tr(result.Content), "", "", false)
	}

	// Save PDF
	savedPath := filepath.Join(a.outputDir, filename)
	if err := pdf.OutputFileAndClose(savedPath); err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to save PDF: %w", err)
	}

	return types.ActionResult{
		Result: fmt.Sprintf("PDF generated and saved to: %s", savedPath),
		Metadata: map[string]interface{}{
			MetadataPDFs: []string{savedPath},
		},
	}, nil
}

func (a *GenPDFAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "generate_pdf",
		Description: "Generate a PDF document from text content. The PDF is saved locally and can be sent to the user by connectors.",
		Properties: map[string]jsonschema.Definition{
			"title": {
				Type:        jsonschema.String,
				Description: "Title of the PDF document",
			},
			"content": {
				Type:        jsonschema.String,
				Description: "Text or Markdown content to include in the PDF (headings, bold, lists, code blocks, etc. are rendered)",
			},
			"filename": {
				Type:        jsonschema.String,
				Description: "Optional custom filename (extension is optional - .pdf will be automatically added if missing)",
			},
		},
		Required: []string{"content"},
	}
}

func (a *GenPDFAction) Plannable() bool {
	return true
}

// GenPDFConfigMeta returns the metadata for GenPDF action configuration fields.
func GenPDFConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "outputDir",
			Label:    "Output directory",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Directory where generated PDF files are saved",
		},
		{
			Name:         "cleanOnStart",
			Label:        "Clean output directory on start",
			Type:         config.FieldTypeCheckbox,
			DefaultValue: false,
			HelpText:     "If enabled, clear the output directory when the action is loaded",
		},
	}
}
