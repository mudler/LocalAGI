package webui

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/mudler/LocalAGI/services/skills"
	"github.com/mudler/xlog"
	skilldomain "github.com/mudler/skillserver/pkg/domain"
	skillgit "github.com/mudler/skillserver/pkg/git"
)

type skillResponse struct {
	Name          string            `json:"name"`
	Content       string            `json:"content"`
	Description   string            `json:"description,omitempty"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
	ReadOnly      bool              `json:"readOnly"`
}

type createSkillRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
}

type updateSkillRequest struct {
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
}

func skillToResponse(s skilldomain.Skill) skillResponse {
	out := skillResponse{Name: s.Name, Content: s.Content, ReadOnly: s.ReadOnly}
	if s.Metadata != nil {
		out.Description = s.Metadata.Description
		out.License = s.Metadata.License
		out.Compatibility = s.Metadata.Compatibility
		out.Metadata = s.Metadata.Metadata
		out.AllowedTools = s.Metadata.AllowedTools
	}
	return out
}

func (a *App) skillsSvc() *skills.Service {
	if a.config == nil {
		return nil
	}
	return a.config.SkillsService
}

func skillsUnavailable(c *fiber.Ctx) error {
	return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "skills service not available"})
}

func skillsNoDir(c *fiber.Ctx) error {
	return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "skills directory not configured"})
}

// decodeSkillNameParam decodes a URL-encoded skill name (e.g. repo%2Fskill -> repo/skill).
func decodeSkillNameParam(raw string) string {
	if raw == "" {
		return ""
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

func (a *App) GetSkillsConfig(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	return c.JSON(fiber.Map{"skills_dir": svc.GetSkillsDir()})
}

func (a *App) ListSkills(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		if mgr == nil {
			return c.Status(http.StatusOK).JSON([]skillResponse{})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	list, err := mgr.ListSkills()
	if err != nil {
		xlog.Error("[skills] ListSkills: mgr.ListSkills failed", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	out := make([]skillResponse, len(list))
	for i, s := range list {
		out[i] = skillToResponse(s)
	}
	return c.JSON(out)
}

func (a *App) SearchSkills(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	q := c.Query("q")
	if q == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	list, err := mgr.SearchSkills(q)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	out := make([]skillResponse, len(list))
	for i, s := range list {
		out[i] = skillToResponse(s)
	}
	return c.JSON(out)
}

func (a *App) GetSkill(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	name := decodeSkillNameParam(c.Params("name"))
	skill, err := mgr.ReadSkill(name)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	return c.JSON(skillToResponse(*skill))
}

func (a *App) CreateSkill(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	fsManager, ok := mgr.(*skilldomain.FileSystemManager)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "unsupported manager type"})
	}
	var req createSkillRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if err := skilldomain.ValidateSkillName(req.Name); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Description == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "description is required"})
	}
	if len(req.Description) > 1024 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "description must be 1-1024 characters"})
	}
	if req.Compatibility != "" && len(req.Compatibility) > 500 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "compatibility must be max 500 characters"})
	}
	skillsDir := fsManager.GetSkillsDir()
	skillDir := filepath.Join(skillsDir, req.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n", req.Name, req.Description)
	if req.License != "" {
		frontmatter += fmt.Sprintf("license: %s\n", req.License)
	}
	if req.Compatibility != "" {
		frontmatter += fmt.Sprintf("compatibility: %s\n", req.Compatibility)
	}
	if len(req.Metadata) > 0 {
		frontmatter += "metadata:\n"
		for k, v := range req.Metadata {
			frontmatter += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	if req.AllowedTools != "" {
		frontmatter += fmt.Sprintf("allowed-tools: %s\n", req.AllowedTools)
	}
	frontmatter += "---\n\n"
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMdPath, []byte(frontmatter+req.Content), 0644); err != nil {
		os.RemoveAll(skillDir)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := mgr.RebuildIndex(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rebuild index"})
	}
	skill, err := mgr.ReadSkill(req.Name)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read created skill"})
	}
	return c.Status(http.StatusCreated).JSON(skillToResponse(*skill))
}

func (a *App) UpdateSkill(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	fsManager, ok := mgr.(*skilldomain.FileSystemManager)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "unsupported manager type"})
	}
	name := decodeSkillNameParam(c.Params("name"))
	existing, err := mgr.ReadSkill(name)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	if existing.ReadOnly {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "cannot update read-only skill from git repository"})
	}
	var req updateSkillRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Description == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "description is required"})
	}
	if len(req.Description) > 1024 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "description must be 1-1024 characters"})
	}
	if req.Compatibility != "" && len(req.Compatibility) > 500 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "compatibility must be max 500 characters"})
	}
	skillDir := filepath.Join(fsManager.GetSkillsDir(), name)
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n", name, req.Description)
	if req.License != "" {
		frontmatter += fmt.Sprintf("license: %s\n", req.License)
	}
	if req.Compatibility != "" {
		frontmatter += fmt.Sprintf("compatibility: %s\n", req.Compatibility)
	}
	if len(req.Metadata) > 0 {
		frontmatter += "metadata:\n"
		for k, v := range req.Metadata {
			frontmatter += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	if req.AllowedTools != "" {
		frontmatter += fmt.Sprintf("allowed-tools: %s\n", req.AllowedTools)
	}
	frontmatter += "---\n\n"
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMdPath, []byte(frontmatter+req.Content), 0644); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := mgr.RebuildIndex(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rebuild index"})
	}
	skill, _ := mgr.ReadSkill(name)
	return c.JSON(skillToResponse(*skill))
}

func (a *App) DeleteSkill(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	fsManager, ok := mgr.(*skilldomain.FileSystemManager)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "unsupported manager type"})
	}
	name := decodeSkillNameParam(c.Params("name"))
	existing, err := mgr.ReadSkill(name)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	if existing.ReadOnly {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "cannot delete read-only skill from git repository"})
	}
	skillDir := filepath.Join(fsManager.GetSkillsDir(), name)
	if err := os.RemoveAll(skillDir); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := mgr.RebuildIndex(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rebuild index"})
	}
	return c.SendStatus(http.StatusNoContent)
}

func (a *App) ExportSkill(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	fsManager, ok := mgr.(*skilldomain.FileSystemManager)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "unsupported manager type"})
	}
	rawName := strings.TrimPrefix(c.Params("*"), "/")
	if rawName == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "skill name required"})
	}
	name := decodeSkillNameParam(rawName)
	skill, err := mgr.ReadSkill(name)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	archiveData, err := skilldomain.ExportSkill(skill.ID, fsManager.GetSkillsDir())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set("Content-Type", "application/gzip")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.gz\"", name))
	return c.Send(archiveData)
}

func (a *App) ImportSkill(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	fsManager, ok := mgr.(*skilldomain.FileSystemManager)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "unsupported manager type"})
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	src, err := file.Open()
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "failed to open uploaded file"})
	}
	defer src.Close()
	const maxArchiveSize = 50 * 1024 * 1024
	archiveData := make([]byte, file.Size)
	if file.Size > maxArchiveSize {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "archive too large"})
	}
	n, err := io.ReadFull(src, archiveData)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "failed to read file"})
	}
	archiveData = archiveData[:n]
	skillName, err := skilldomain.ImportSkill(archiveData, fsManager.GetSkillsDir())
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := mgr.RebuildIndex(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rebuild index"})
	}
	skill, _ := mgr.ReadSkill(skillName)
	return c.Status(http.StatusCreated).JSON(skillToResponse(*skill))
}

func (a *App) ListSkillResources(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	skillName := decodeSkillNameParam(c.Params("name"))
	skill, err := mgr.ReadSkill(skillName)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	resources, err := mgr.ListSkillResources(skill.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	scripts := []map[string]interface{}{}
	references := []map[string]interface{}{}
	assets := []map[string]interface{}{}
	for _, res := range resources {
		m := map[string]interface{}{
			"path":      res.Path,
			"name":      res.Name,
			"size":      res.Size,
			"mime_type": res.MimeType,
			"readable":  res.Readable,
			"modified":  res.Modified.Format("2006-01-02T15:04:05Z07:00"),
		}
		switch res.Type {
		case skilldomain.ResourceTypeScript:
			scripts = append(scripts, m)
		case skilldomain.ResourceTypeReference:
			references = append(references, m)
		case skilldomain.ResourceTypeAsset:
			assets = append(assets, m)
		}
	}
	return c.JSON(fiber.Map{"scripts": scripts, "references": references, "assets": assets, "readOnly": skill.ReadOnly})
}

func (a *App) GetSkillResource(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	skillName := decodeSkillNameParam(c.Params("name"))
	resourcePath := c.Params("*")
	if resourcePath == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "resource path is required"})
	}
	skill, err := mgr.ReadSkill(skillName)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	info, err := mgr.GetSkillResourceInfo(skill.ID, resourcePath)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "resource not found"})
	}
	content, err := mgr.ReadSkillResource(skill.ID, resourcePath)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if c.Query("encoding") == "base64" || !info.Readable {
		return c.JSON(fiber.Map{"content": content.Content, "encoding": content.Encoding, "mime_type": content.MimeType, "size": content.Size})
	}
	c.Set("Content-Type", content.MimeType)
	return c.SendString(content.Content)
}

func (a *App) CreateSkillResource(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	skillName := decodeSkillNameParam(c.Params("name"))
	skill, err := mgr.ReadSkill(skillName)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	if skill.ReadOnly {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "cannot add resources to read-only skill"})
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	path := c.FormValue("path")
	if path == "" {
		path = file.Filename
	}
	if err := skilldomain.ValidateResourcePath(path); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	fullPath := filepath.Join(skill.SourcePath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	src, err := file.Open()
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"path": path})
}

func (a *App) UpdateSkillResource(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	skillName := decodeSkillNameParam(c.Params("name"))
	resourcePath := c.Params("*")
	if resourcePath == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "resource path is required"})
	}
	skill, err := mgr.ReadSkill(skillName)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	if skill.ReadOnly {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "cannot update resources in read-only skill"})
	}
	if err := skilldomain.ValidateResourcePath(resourcePath); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	fullPath := filepath.Join(skill.SourcePath, resourcePath)
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := os.WriteFile(fullPath, []byte(body.Content), 0644); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(http.StatusNoContent)
}

func (a *App) DeleteSkillResource(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		return skillsNoDir(c)
	}
	skillName := decodeSkillNameParam(c.Params("name"))
	resourcePath := c.Params("*")
	if resourcePath == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "resource path is required"})
	}
	skill, err := mgr.ReadSkill(skillName)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}
	if skill.ReadOnly {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "cannot delete resources from read-only skill"})
	}
	if err := skilldomain.ValidateResourcePath(resourcePath); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	fullPath := filepath.Join(skill.SourcePath, resourcePath)
	if err := os.Remove(fullPath); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(http.StatusNoContent)
}

// Git repos: list, add, update, delete, sync, toggle (using ConfigManager in skills dir)
func (a *App) ListGitRepos(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	dir := svc.GetSkillsDir()
	if dir == "" {
		return c.Status(http.StatusOK).JSON([]gitRepoResponse{})
	}
	cm := skillgit.NewConfigManager(dir)
	repos, err := cm.LoadConfig()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	out := make([]gitRepoResponse, len(repos))
	for i, r := range repos {
		out[i] = gitRepoResponse{ID: r.ID, URL: r.URL, Name: r.Name, Enabled: r.Enabled}
	}
	return c.JSON(out)
}

type gitRepoResponse struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (a *App) AddGitRepo(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	dir := svc.GetSkillsDir()
	if dir == "" {
		return skillsNoDir(c)
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil || req.URL == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "URL is required"})
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "git@") {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid URL format"})
	}
	cm := skillgit.NewConfigManager(dir)
	repos, err := cm.LoadConfig()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	for _, r := range repos {
		if r.URL == req.URL {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "repository already exists"})
		}
	}
	newRepo := skillgit.GitRepoConfig{
		ID:      skillgit.GenerateID(req.URL),
		URL:     req.URL,
		Name:    skillgit.ExtractRepoName(req.URL),
		Enabled: true,
	}
	repos = append(repos, newRepo)
	if err := cm.SaveConfig(repos); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	// Do not invalidate here: the new repo is not cloned yet. Keep the current manager
	// so ListSkills returns immediately and the sync goroutine gets the cache without contention.
	urlToSync := req.URL
	xlog.Info("[skills] AddGitRepo: repo saved, starting background sync", "url", urlToSync)
	go func() {
		xlog.Info("[skills] background sync: started", "url", urlToSync)
		mgr, err := svc.GetManager()
		if err != nil || mgr == nil {
			xlog.Error("[skills] background sync: GetManager failed", "url", urlToSync, "error", err)
			return
		}
		xlog.Info("[skills] background sync: got manager, running syncer", "url", urlToSync)
		syncer := skillgit.NewGitSyncer(dir, []string{urlToSync}, mgr.RebuildIndex)
		if err := syncer.Start(); err != nil {
			xlog.Error("[skills] background sync: sync failed", "url", urlToSync, "error", err)
			svc.RefreshManagerFromConfig()
			return
		}
		syncer.Stop()
		svc.RefreshManagerFromConfig()
		xlog.Info("[skills] background sync: finished", "url", urlToSync)
	}()
	xlog.Info("[skills] AddGitRepo: returning 201 (sync in progress)")
	return c.Status(http.StatusCreated).JSON(gitRepoResponse{ID: newRepo.ID, URL: newRepo.URL, Name: newRepo.Name, Enabled: newRepo.Enabled})
}

func (a *App) UpdateGitRepo(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	dir := svc.GetSkillsDir()
	if dir == "" {
		return skillsNoDir(c)
	}
	id := c.Params("id")
	var req struct {
		URL     string `json:"url"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	cm := skillgit.NewConfigManager(dir)
	repos, err := cm.LoadConfig()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	var found int
	for i, r := range repos {
		if r.ID == id {
			found = i
			if req.URL != "" {
				repos[i].URL = req.URL
				repos[i].Name = skillgit.ExtractRepoName(req.URL)
			}
			if req.Enabled != nil {
				repos[i].Enabled = *req.Enabled
			}
			break
		}
	}
	if found >= len(repos) || repos[found].ID != id {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "repository not found"})
	}
	if err := cm.SaveConfig(repos); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	svc.RefreshManagerFromConfig()
	return c.JSON(gitRepoResponse{ID: repos[found].ID, URL: repos[found].URL, Name: repos[found].Name, Enabled: repos[found].Enabled})
}

func (a *App) DeleteGitRepo(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	dir := svc.GetSkillsDir()
	if dir == "" {
		return skillsNoDir(c)
	}
	id := c.Params("id")
	cm := skillgit.NewConfigManager(dir)
	repos, err := cm.LoadConfig()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	var newRepos []skillgit.GitRepoConfig
	var repoName string
	for _, r := range repos {
		if r.ID == id {
			repoName = r.Name
		} else {
			newRepos = append(newRepos, r)
		}
	}
	if len(newRepos) == len(repos) {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "repository not found"})
	}
	if err := cm.SaveConfig(newRepos); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if repoName != "" {
		repoDir := filepath.Join(dir, repoName)
		if err := os.RemoveAll(repoDir); err != nil {
			xlog.Warn("[skills] DeleteGitRepo: failed to remove repo directory", "dir", repoDir, "error", err)
		}
	}
	svc.RefreshManagerFromConfig()
	return c.SendStatus(http.StatusNoContent)
}

func (a *App) SyncGitRepo(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	dir := svc.GetSkillsDir()
	if dir == "" {
		return skillsNoDir(c)
	}
	id := c.Params("id")
	cm := skillgit.NewConfigManager(dir)
	repos, err := cm.LoadConfig()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	var url string
	for _, r := range repos {
		if r.ID == id {
			url = r.URL
			break
		}
	}
	if url == "" {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "repository not found"})
	}
	xlog.Info("[skills] SyncGitRepo: requested", "id", id, "url", url)
	mgr, err := svc.GetManager()
	if err != nil || mgr == nil {
		xlog.Error("[skills] SyncGitRepo: GetManager failed", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "manager not ready"})
	}
	go func() {
		xlog.Info("[skills] SyncGitRepo: background sync started", "id", id, "url", url)
		syncer := skillgit.NewGitSyncer(dir, []string{url}, mgr.RebuildIndex)
		if err := syncer.Start(); err != nil {
			xlog.Error("[skills] SyncGitRepo: background sync failed", "id", id, "error", err)
			svc.RefreshManagerFromConfig()
			return
		}
		syncer.Stop()
		svc.RefreshManagerFromConfig()
		xlog.Info("[skills] SyncGitRepo: background sync finished", "id", id)
	}()
	xlog.Info("[skills] SyncGitRepo: returning 200 (sync in progress)")
	return c.JSON(fiber.Map{"status": "ok", "message": "Sync started in background"})
}

func (a *App) ToggleGitRepo(c *fiber.Ctx) error {
	svc := a.skillsSvc()
	if svc == nil {
		return skillsUnavailable(c)
	}
	dir := svc.GetSkillsDir()
	if dir == "" {
		return skillsNoDir(c)
	}
	id := c.Params("id")
	cm := skillgit.NewConfigManager(dir)
	repos, err := cm.LoadConfig()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	for i, r := range repos {
		if r.ID == id {
			repos[i].Enabled = !repos[i].Enabled
			if err := cm.SaveConfig(repos); err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			svc.RefreshManagerFromConfig()
			return c.JSON(gitRepoResponse{ID: repos[i].ID, URL: repos[i].URL, Name: repos[i].Name, Enabled: repos[i].Enabled})
		}
	}
	return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "repository not found"})
}
