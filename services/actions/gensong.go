package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	MetadataSongs = "songs_paths"
)

// audioExtensionFromContentType returns a file extension for common audio MIME types.
// It strips parameters (e.g. "audio/flac; rate=44100" -> "flac").
func audioExtensionFromContentType(contentType string) string {
	mediaType, _, _ := strings.Cut(strings.TrimSpace(contentType), ";")
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	switch mediaType {
	case "audio/flac":
		return "flac"
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return "wav"
	case "audio/ogg":
		return "ogg"
	case "audio/webm":
		return "webm"
	default:
		return ""
	}
}

// audioExtensionFromTag uses github.com/dhowden/tag to identify format from audio bytes.
// Identify works on raw audio (e.g. FLAC without vorbis comments) and returns FileType.
func audioExtensionFromTag(data []byte) string {
	if len(data) < 11 {
		return ""
	}
	r := bytes.NewReader(data)
	_, fileType, err := tag.Identify(r)
	if err != nil || fileType == tag.UnknownFileType {
		return ""
	}
	switch fileType {
	case tag.FLAC:
		return "flac"
	case tag.MP3:
		return "mp3"
	case tag.OGG:
		return "ogg"
	case tag.M4A:
		return "m4a"
	case tag.M4B:
		return "m4b"
	case tag.M4P:
		return "m4p"
	case tag.ALAC:
		return "m4a"
	case tag.DSF:
		return "dsf"
	default:
		return ""
	}
}

// soundRequest matches LocalAI /sound endpoint (ACE-Step advanced mode) request body.
// See: https://localai.io/features/text-to-audio/
type soundRequest struct {
	Model           string   `json:"model"`
	Caption         string   `json:"caption"`
	Lyrics          string   `json:"lyrics,omitempty"`
	BPM             *int     `json:"bpm,omitempty"`
	Keyscale        string   `json:"keyscale,omitempty"`
	Language        string   `json:"language,omitempty"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
}

func NewGenSong(config map[string]string) *GenSongAction {
	model := config["model"]
	if model == "" {
		model = "ace-step-turbo"
	}
	a := &GenSongAction{
		apiURL:       strings.TrimSuffix(config["apiURL"], "/"),
		apiKey:       config["apiKey"],
		outputDir:    config["outputDir"],
		model:        model,
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

type GenSongAction struct {
	apiURL       string
	apiKey       string
	outputDir    string
	model        string
	cleanOnStart bool
}

func (a *GenSongAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Caption  string   `json:"caption"`
		Lyrics   string   `json:"lyrics"`
		BPM      *int     `json:"bpm"`
		Keyscale string   `json:"keyscale"`
		Language string   `json:"language"`
		Duration *float64 `json:"duration_seconds"`
		Model    string   `json:"model"`
	}{}
	if err := params.Unmarshal(&result); err != nil {
		return types.ActionResult{}, err
	}

	if result.Caption == "" {
		return types.ActionResult{}, fmt.Errorf("caption is required")
	}

	if a.outputDir == "" {
		return types.ActionResult{}, fmt.Errorf("outputDir is required for generate_song (configure the action with an output directory)")
	}

	reqBody := soundRequest{
		Model:           a.model,
		Caption:         result.Caption,
		Lyrics:          result.Lyrics,
		Keyscale:        result.Keyscale,
		Language:        result.Language,
		DurationSeconds: result.Duration,
		BPM:             result.BPM,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return types.ActionResult{}, err
	}

	url := a.apiURL + "/sound"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return types.ActionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("xi-api-key", a.apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return types.ActionResult{Result: "Failed to generate song: " + err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return types.ActionResult{}, fmt.Errorf("sound endpoint failed: %s: %s", resp.Status, string(msg))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.ActionResult{}, err
	}
	if len(audioBytes) == 0 {
		return types.ActionResult{}, fmt.Errorf("no audio data returned")
	}

	ext := audioExtensionFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		ext = audioExtensionFromTag(audioBytes)
	}
	if ext == "" {
		ext = "flac" // default when unknown (e.g. ACE-Step)
	}

	filename := fmt.Sprintf("song_%d.%s", time.Now().UnixNano(), ext)
	savedPath := filepath.Join(a.outputDir, filename)
	if err := os.WriteFile(savedPath, audioBytes, 0644); err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to save song: %w", err)
	}

	return types.ActionResult{
		Result: fmt.Sprintf("The song was generated and saved to: %s", savedPath),
		Metadata: map[string]interface{}{
			MetadataSongs: []string{savedPath},
		},
	}, nil
}

func (a *GenSongAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "generate_song",
		Description: "Generate a song or music track using LocalAI /sound endpoint (ACE-Step advanced mode). Uses caption, optional lyrics, BPM, key scale, language and duration. The file is saved locally and can be sent to the user by connectors.",
		Properties: map[string]jsonschema.Definition{
			"caption": {
				Type:        jsonschema.String,
				Description: "Description of the song or music to generate (e.g. 'A funky Japanese disco track').",
			},
			"lyrics": {
				Type:        jsonschema.String,
				Description: "Lyrics or structure (e.g. '[Verse 1]\\n...').",
			},
			"bpm": {
				Type:        jsonschema.Integer,
				Description: "Beats per minute (e.g. 120). Optional.",
			},
			"keyscale": {
				Type:        jsonschema.String,
				Description: "Key and scale (e.g. 'Ab major'). Optional.",
			},
			"language": {
				Type:        jsonschema.String,
				Description: "Language code for vocals (e.g. 'ja', 'en'). Optional.",
			},
			"duration_seconds": {
				Type:        jsonschema.Number,
				Description: "Duration of the generated audio in seconds (e.g. 225). Optional.",
			},
			"model": {
				Type:        jsonschema.String,
				Description: "Model name (e.g. ace-step-turbo). Optional; uses action config default if omitted.",
			},
		},
		Required: []string{"caption"},
	}
}

func (a *GenSongAction) Plannable() bool {
	return true
}

// GenSongConfigMeta returns the metadata for GenSong action configuration fields.
func GenSongConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:         "apiURL",
			Label:        "API URL",
			Type:         config.FieldTypeText,
			Required:     true,
			DefaultValue: "http://localhost:8080",
			HelpText:     "LocalAI base URL (e.g. http://localhost:8080) for /sound endpoint",
		},
		{
			Name:         "model",
			Label:        "Model",
			Type:         config.FieldTypeText,
			Required:     false,
			DefaultValue: "ace-step-turbo",
			HelpText:     "Default model for sound generation (e.g. ace-step-turbo)",
		},
		{
			Name:     "apiKey",
			Label:    "API Key",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Optional API key if the endpoint requires authentication",
		},
		{
			Name:     "outputDir",
			Label:    "Output directory",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Directory where generated song files are saved (required for connectors to send files)",
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
