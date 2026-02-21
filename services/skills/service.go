package skills

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/xlog"

	skilldomain "github.com/mudler/skillserver/pkg/domain"
	skillgit "github.com/mudler/skillserver/pkg/git"
	skillmcp "github.com/mudler/skillserver/pkg/mcp"
)

// SkillsDirName is the subdirectory under state dir where skills are stored
const SkillsDirName = "skills"

// Service manages the skills directory (fixed at stateDir/skills), lazy SkillManager, dynamic prompt, and in-process MCP session
type Service struct {
	stateDir string
	mu       sync.Mutex
	manager  skilldomain.SkillManager
	mcpSrv   *skillmcp.Server
	session  *mcp.ClientSession
}

// NewService creates a skills service. Skills are stored under stateDir/skills.
func NewService(stateDir string) (*Service, error) {
	return &Service{
		stateDir: stateDir,
	}, nil
}

// GetSkillsDir returns the skills directory path (always stateDir/skills)
func (s *Service) GetSkillsDir() string {
	return filepath.Join(s.stateDir, SkillsDirName)
}

// InvalidateManager clears the cached manager (e.g. after git repo config change) so next use reloads from disk
func (s *Service) InvalidateManager() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manager = nil
	s.mcpSrv = nil
	s.session = nil
}

// getManagerLocked returns the SkillManager, lazily creating it; caller must hold s.mu
func (s *Service) getManagerLocked() (skilldomain.SkillManager, error) {
	if s.manager != nil {
		return s.manager, nil
	}
	skillsDir := s.GetSkillsDir()
	gitRepos := []string{}
	cm := skillgit.NewConfigManager(skillsDir)
	repos, err := cm.LoadConfig()
	if err != nil {
		xlog.Warn("Could not load git-repos config for skills", "error", err)
	} else {
		for _, r := range repos {
			if r.Enabled && r.Name != "" {
				gitRepos = append(gitRepos, r.Name)
			}
		}
	}
	mgr, err := skilldomain.NewFileSystemManager(skillsDir, gitRepos)
	if err != nil {
		return nil, err
	}
	s.manager = mgr
	return s.manager, nil
}

// GetManager returns the SkillManager if the skills dir is set, otherwise nil
func (s *Service) GetManager() (skilldomain.SkillManager, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getManagerLocked()
}

// GetSkillsPrompt returns a DynamicPrompt that injects the available skills XML (or nil if no manager)
func (s *Service) GetSkillsPrompt() (agent.DynamicPrompt, error) {
	s.mu.Lock()
	mgr, err := s.getManagerLocked()
	s.mu.Unlock()
	if err != nil || mgr == nil {
		return nil, err
	}
	return NewSkillsPrompt(mgr.ListSkills), nil
}

// GetMCPSession returns a shared MCP client session connected to the in-process skillserver (starts on first use)
func (s *Service) GetMCPSession(ctx context.Context) (*mcp.ClientSession, error) {
	s.mu.Lock()
	if s.session != nil {
		sess := s.session
		s.mu.Unlock()
		return sess, nil
	}
	mgr, err := s.getManagerLocked()
	if err != nil || mgr == nil {
		s.mu.Unlock()
		return nil, err
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	s.mcpSrv = skillmcp.NewServer(mgr)
	go func() {
		if err := s.mcpSrv.RunWithTransport(ctx, serverTransport); err != nil && ctx.Err() == nil {
			xlog.Error("Skills MCP server exited", "error", err)
		}
	}()
	client := mcp.NewClient(&mcp.Implementation{Name: "LocalAGI", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.session = session
	s.mu.Unlock()
	return session, nil
}
