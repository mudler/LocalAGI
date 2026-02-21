import { useState, useEffect } from 'react';
import { Link, useOutletContext } from 'react-router-dom';
import { skillsApi } from '../utils/api';

function Skills() {
  const [skills, setSkills] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const [unavailable, setUnavailable] = useState(false);
  const [showGitRepos, setShowGitRepos] = useState(false);
  const [gitRepos, setGitRepos] = useState([]);
  const [gitRepoUrl, setGitRepoUrl] = useState('');
  const [gitReposLoading, setGitReposLoading] = useState(false);
  const [gitReposAction, setGitReposAction] = useState(null);
  const { showToast } = useOutletContext();

  const fetchSkills = async () => {
    setLoading(true);
    setUnavailable(false);
    const timeoutMs = 15000;
    const withTimeout = (p) =>
      Promise.race([
        p,
        new Promise((_, reject) =>
          setTimeout(() => reject(new Error('Request timed out')), timeoutMs)
        ),
      ]);
    try {
      if (searchQuery.trim()) {
        const data = await withTimeout(skillsApi.search(searchQuery.trim()));
        setSkills(Array.isArray(data) ? data : []);
      } else {
        const data = await withTimeout(skillsApi.list());
        setSkills(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      if (err.message?.includes('503') || err.message?.includes('skills')) {
        setUnavailable(true);
        setSkills([]);
      } else {
        showToast(err.message || 'Failed to load skills', 'error');
        setSkills([]);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    document.title = 'Skills - LocalAGI';
  }, []);

  useEffect(() => {
    fetchSkills();
  }, [searchQuery]);

  const deleteSkill = async (name) => {
    if (!confirm(`Delete skill "${name}"?`)) return;
    try {
      await skillsApi.delete(name);
      showToast('Skill deleted', 'success');
      fetchSkills();
    } catch (err) {
      showToast(err.message || 'Failed to delete skill', 'error');
    }
  };

  const exportSkill = async (name) => {
    try {
      const url = skillsApi.exportUrl(name);
      const res = await fetch(url, { credentials: 'same-origin' });
      if (!res.ok) throw new Error(res.statusText || 'Export failed');
      const blob = await res.blob();
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = `${name.replace(/\//g, '-')}.tar.gz`;
      a.click();
      URL.revokeObjectURL(a.href);
      showToast('Export started', 'success');
    } catch (err) {
      showToast(err.message || 'Export failed', 'error');
    }
  };

  const handleImport = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setImporting(true);
    try {
      await skillsApi.import(file);
      showToast('Skill imported', 'success');
      fetchSkills();
    } catch (err) {
      showToast(err.message || 'Import failed', 'error');
    } finally {
      setImporting(false);
      e.target.value = '';
    }
  };

  const loadGitRepos = async () => {
    setGitReposLoading(true);
    try {
      const list = await skillsApi.listGitRepos();
      setGitRepos(Array.isArray(list) ? list : []);
    } catch (err) {
      showToast(err.message || 'Failed to load Git repos', 'error');
      setGitRepos([]);
    } finally {
      setGitReposLoading(false);
    }
  };

  useEffect(() => {
    if (showGitRepos) loadGitRepos();
  }, [showGitRepos]);

  const addGitRepo = async (e) => {
    e.preventDefault();
    const url = gitRepoUrl.trim();
    if (!url) return;
    setGitReposAction('add');
    try {
      await skillsApi.addGitRepo(url);
      setGitRepoUrl('');
      await loadGitRepos();
      fetchSkills();
      showToast('Git repo added and syncing', 'success');
    } catch (err) {
      showToast(err.message || 'Failed to add repo', 'error');
    } finally {
      setGitReposAction(null);
    }
  };

  const syncGitRepo = async (id) => {
    setGitReposAction(id);
    try {
      await skillsApi.syncGitRepo(id);
      await loadGitRepos();
      fetchSkills();
      showToast('Repo synced', 'success');
    } catch (err) {
      showToast(err.message || 'Sync failed', 'error');
    } finally {
      setGitReposAction(null);
    }
  };

  const toggleGitRepo = async (id) => {
    try {
      await skillsApi.toggleGitRepo(id);
      await loadGitRepos();
      fetchSkills();
      showToast('Repo toggled', 'success');
    } catch (err) {
      showToast(err.message || 'Toggle failed', 'error');
    }
  };

  const deleteGitRepo = async (id) => {
    if (!confirm('Remove this Git repository? Skills from it will no longer be available.')) return;
    try {
      await skillsApi.deleteGitRepo(id);
      await loadGitRepos();
      fetchSkills();
      showToast('Repo removed', 'success');
    } catch (err) {
      showToast(err.message || 'Remove failed', 'error');
    }
  };

  if (unavailable) {
    return (
      <div className="page skills-page">
        <header className="page-header">
          <h1>Skills</h1>
          <p className="page-description">Skills service is not available or the index is rebuilding. Try again in a moment.</p>
          <button type="button" className="btn btn-primary" onClick={() => { setUnavailable(false); fetchSkills(); }}>
            Retry
          </button>
        </header>
      </div>
    );
  }

  return (
    <div className="page skills-page">
      <header className="page-header">
        <div>
          <h1>Skills</h1>
          <p className="page-description">Manage agent skills (reusable instructions and resources). Skills are stored under the state directory. Create or import skills, and enable &quot;Enable Skills&quot; per agent to give them access.</p>
        </div>
        <div className="header-actions" style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap', alignItems: 'center' }}>
          <input
            type="text"
            className="input"
            placeholder="Search skills..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{ width: '220px' }}
          />
          <Link to="/skills/new" className="action-btn success">
            <i className="fas fa-plus" /> New skill
          </Link>
          <label className="action-btn" style={{ margin: 0, cursor: 'pointer' }}>
            <input type="file" accept=".tar.gz" onChange={handleImport} disabled={importing} style={{ display: 'none' }} />
            {importing ? 'Importing...' : <><i className="fas fa-file-import" /> Import</>}
          </label>
          <button type="button" className="action-btn" onClick={() => setShowGitRepos((v) => !v)}>
            <i className="fas fa-code-branch" /> Git Repos
          </button>
        </div>
      </header>

      {showGitRepos && (
        <div className="section-box" style={{ marginBottom: '1.5rem' }}>
          <h2 className="section-title" style={{ marginTop: 0 }}>
            <i className="fas fa-code-branch" /> Git repositories
          </h2>
          <p className="page-description" style={{ marginBottom: '1rem' }}>
            Add Git repositories to pull skills from. Skills will appear in the list after sync.
          </p>
          <form onSubmit={addGitRepo} style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', marginBottom: '1rem' }}>
            <input
              type="url"
              className="input"
              placeholder="https://github.com/user/repo or git@github.com:user/repo.git"
              value={gitRepoUrl}
              onChange={(e) => setGitRepoUrl(e.target.value)}
              style={{ flex: '1', minWidth: '200px' }}
            />
            <button type="submit" className="action-btn success" disabled={gitReposAction === 'add'}>
              {gitReposAction === 'add' ? 'Adding...' : 'Add repo'}
            </button>
          </form>
          {gitReposLoading ? (
            <p>Loading repos...</p>
          ) : gitRepos.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)' }}>No Git repos configured. Add one above.</p>
          ) : (
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {gitRepos.map((r) => (
                <li key={r.id} className="card" style={{ padding: '0.75rem 1rem', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '0.5rem' }}>
                  <div>
                    <span style={{ fontWeight: 600 }}>{r.name || r.url}</span>
                    <span style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', marginLeft: '0.5rem' }}>{r.url}</span>
                    {!r.enabled && <span className="badge" style={{ marginLeft: '0.5rem' }}>Disabled</span>}
                  </div>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button type="button" className="action-btn" onClick={() => syncGitRepo(r.id)} disabled={gitReposAction === r.id}>
                      {gitReposAction === r.id ? 'Syncing...' : <><i className="fas fa-sync-alt" /> Sync</>}
                    </button>
                    <button type="button" className="action-btn" onClick={() => toggleGitRepo(r.id)} title={r.enabled ? 'Disable' : 'Enable'}>
                      <i className={`fas fa-toggle-${r.enabled ? 'on' : 'off'}`} />
                    </button>
                    <button type="button" className="action-btn delete-btn" onClick={() => deleteGitRepo(r.id)} title="Remove repo">
                      <i className="fas fa-trash" />
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {loading ? (
        <p>Loading skills...</p>
      ) : skills.length === 0 ? (
        <div className="card">
          <p>No skills found. Create a skill or import one.</p>
          <Link to="/skills/new" className="action-btn success" style={{ marginTop: '0.5rem' }}>Create skill</Link>
        </div>
      ) : (
        <div className="skills-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1rem' }}>
          {skills.map((s) => (
            <div key={s.name} className="card" style={{ padding: '1rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.5rem' }}>
                <h3 style={{ margin: 0, fontSize: '1.1rem' }}>{s.name}</h3>
                {s.readOnly && <span className="badge" style={{ fontSize: '0.75rem' }}>Read-only</span>}
              </div>
              <p style={{ margin: '0 0 0.75rem 0', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                {s.description || 'No description'}
              </p>
              <div className="agent-table-actions" style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                {!s.readOnly && (
                  <Link to={`/skills/edit/${encodeURIComponent(s.name)}`} className="action-btn" title="Edit skill">
                    <i className="fas fa-edit" /> Edit
                  </Link>
                )}
                {!s.readOnly && (
                  <button type="button" className="action-btn delete-btn" onClick={() => deleteSkill(s.name)} title="Delete skill">
                    <i className="fas fa-trash" /> Delete
                  </button>
                )}
                <button type="button" className="action-btn" onClick={() => exportSkill(s.name)} title="Export as .tar.gz">
                  <i className="fas fa-download" /> Export
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default Skills;
