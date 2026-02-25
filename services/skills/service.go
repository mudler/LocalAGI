package skills

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/xlog"

	skilldomain "github.com/mudler/skillserver/pkg/domain"
	skillgit "github.com/mudler/skillserver/pkg/git"
	skillmcp "github.com/mudler/skillserver/pkg/mcp"
)

// SkillsDirName is the subdirectory under state dir where skills are stored
const SkillsDirName = "skills"

// Service manages the skills directory (fixed at stateDir/skills), lazy SkillManager, dynamic prompt, and in-process MCP session
type Service struct {
	stateDir  string
	mu        sync.Mutex
	createMu  sync.Mutex // serializes manager creation so only one createManager() runs at a time
	manager   skilldomain.SkillManager
	mcpSrv    *skillmcp.Server
	session   *mcp.ClientSession
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

// RefreshManagerFromConfig updates the existing manager's git repo list and rebuilds the index
// (same as skillserver: UpdateGitRepos + RebuildIndex in place). Does nothing if no manager exists yet.
// Call this when git repo config changes instead of invalidating; avoids blocking ListSkills on full recreate.
func (s *Service) RefreshManagerFromConfig() {
	skillsDir := s.GetSkillsDir()
	cm := skillgit.NewConfigManager(skillsDir)
	repos, err := cm.LoadConfig()
	if err != nil {
		xlog.Warn("[skills] RefreshManagerFromConfig: could not load config", "error", err)
		return
	}
	gitRepoNames := make([]string, 0, len(repos))
	for _, r := range repos {
		if r.Enabled && r.Name != "" {
			gitRepoNames = append(gitRepoNames, r.Name)
		}
	}
	s.mu.Lock()
	mgr := s.manager
	s.mu.Unlock()
	if mgr == nil {
		return
	}
	if fm, ok := mgr.(*skilldomain.FileSystemManager); ok {
		fm.UpdateGitRepos(gitRepoNames)
		if err := mgr.RebuildIndex(); err != nil {
			xlog.Warn("[skills] RefreshManagerFromConfig: RebuildIndex failed", "error", err)
		}
	}
}

// createManager builds a new SkillManager (reads config and calls NewFileSystemManager).
// Must be called without holding s.mu because NewFileSystemManager runs RebuildIndex() which is slow.
func (s *Service) createManager() (skilldomain.SkillManager, error) {
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
	return mgr, nil
}

// GetManager returns the SkillManager if the skills dir is set, otherwise nil.
// Manager creation is serialized (createMu) so only one createManager() runs at a time,
// avoiding concurrent RebuildIndex and filesystem contention.
func (s *Service) GetManager() (skilldomain.SkillManager, error) {
	s.mu.Lock()
	if s.manager != nil {
		mgr := s.manager
		s.mu.Unlock()
		return mgr, nil
	}
	s.mu.Unlock()

	s.createMu.Lock()
	defer s.createMu.Unlock()

	s.mu.Lock()
	if s.manager != nil {
		mgr := s.manager
		s.mu.Unlock()
		return mgr, nil
	}
	s.mu.Unlock()

	mgr, err := s.createManager()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.manager = mgr
	s.mu.Unlock()
	return mgr, nil
}

// GetSkillsPrompt returns a DynamicPrompt that injects the available skills XML (or nil if no manager).
// When config is non-nil and config.SkillsPrompt is set, that text is used as the intro; otherwise the default intro is used.
func (s *Service) GetSkillsPrompt(config *state.AgentConfig) (agent.DynamicPrompt, error) {
	mgr, err := s.GetManager()
	if err != nil || mgr == nil {
		return nil, err
	}
	customTemplate := ""
	if config != nil && config.SkillsPrompt != "" {
		customTemplate = config.SkillsPrompt
	}
	return NewSkillsPrompt(mgr.ListSkills, customTemplate), nil
}

// GetMCPSession returns a shared MCP client session connected to the in-process skillserver (starts on first use)
func (s *Service) GetMCPSession(ctx context.Context) (*mcp.ClientSession, error) {
	s.mu.Lock()
	if s.session != nil {
		sess := s.session
		s.mu.Unlock()
		return sess, nil
	}
	s.mu.Unlock()

	mgr, err := s.GetManager()
	if err != nil || mgr == nil {
		return nil, err
	}

	s.mu.Lock()
	if s.session != nil {
		sess := s.session
		s.mu.Unlock()
		return sess, nil
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
