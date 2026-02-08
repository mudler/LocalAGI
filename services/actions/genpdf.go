package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
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
			// log but continue; Run will fail with a clear error when saving
			_ = err
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

	// Ensure filename has .pdf extension
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		filename = filename + ".pdf"
	}

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Add title if provided
	if result.Title != "" {
		pdf.SetFont("Arial", "B", 16)
		pdf.MultiCell(0, 10, result.Title, "", "", false)
		pdf.Ln(5)
	}

	// Add content
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 10, result.Content, "", "", false)

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
				Description: "Text content to include in the PDF document",
			},
			"filename": {
				Type:        jsonschema.String,
				Description: "Optional custom filename (without .pdf extension)",
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
