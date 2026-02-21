import { useState, useEffect } from 'react';
import { useParams, useNavigate, useLocation, Link, useOutletContext } from 'react-router-dom';
import { skillsApi } from '../utils/api';

function SkillEdit() {
  const { name: nameParam } = useParams();
  const location = useLocation();
  const isNew = location.pathname.endsWith('/new');
  const name = nameParam ? decodeURIComponent(nameParam) : undefined;
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    name: '',
    description: '',
    content: '',
    license: '',
    compatibility: '',
    metadata: {},
    allowedTools: '',
  });

  useEffect(() => {
    document.title = isNew ? 'New skill - LocalAGI' : `Edit ${name} - LocalAGI`;
    if (isNew) {
      setLoading(false);
      return;
    }
    if (name) {
      skillsApi.get(name)
        .then((data) => {
          setForm({
            name: data.name || '',
            description: data.description || '',
            content: data.content || '',
            license: data.license || '',
            compatibility: data.compatibility || '',
            metadata: data.metadata || {},
            allowedTools: data['allowed-tools'] || '',
          });
        })
        .catch((err) => {
          showToast(err.message || 'Failed to load skill', 'error');
          navigate('/skills');
        })
        .finally(() => setLoading(false));
    }
  }, [isNew, name, navigate, showToast]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = {
        name: form.name,
        description: form.description,
        content: form.content,
        license: form.license || undefined,
        compatibility: form.compatibility || undefined,
        metadata: Object.keys(form.metadata).length ? form.metadata : undefined,
        'allowed-tools': form.allowedTools || undefined,
      };
      if (isNew) {
        await skillsApi.create(payload);
        showToast('Skill created', 'success');
      } else {
        await skillsApi.update(name, { ...payload, name: undefined });
        showToast('Skill updated', 'success');
      }
      navigate('/skills');
    } catch (err) {
      showToast(err.message || 'Save failed', 'error');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="create-agent-container">
        <div className="loading" style={{ padding: '2rem', textAlign: 'center' }}>
          <div className="loader" />
          <p>Loading skill...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="create-agent-container">
      <header className="page-header">
        <Link to="/skills" className="back-link" style={{ marginBottom: '0.5rem', display: 'inline-block' }}>
          <i className="fas fa-arrow-left" /> Back to skills
        </Link>
        <h1>
          <i className="fas fa-book" /> {isNew ? 'New skill' : `Edit: ${name}`}
        </h1>
      </header>

      <div className="create-agent-content">
        <div className="section-box">
          <h2>
            <i className="fas fa-cog" /> Skill configuration
          </h2>

          <form onSubmit={handleSubmit}>
            <h3 className="section-title">Basic information</h3>
            <div className="form-group">
              <label htmlFor="skill-name">Name (lowercase, hyphens only)</label>
              <input
                id="skill-name"
                type="text"
                className="input"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
                disabled={!isNew}
                placeholder="my-skill"
              />
              {!isNew && <p className="help-text">Name cannot be changed after creation.</p>}
            </div>
            <div className="form-group">
              <label htmlFor="skill-desc">Description (required, 1–1024 chars)</label>
              <textarea
                id="skill-desc"
                className="input"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                required
                maxLength={1024}
                rows={2}
              />
            </div>
            <div className="form-group">
              <label htmlFor="skill-license">License (optional)</label>
              <input
                id="skill-license"
                type="text"
                className="input"
                value={form.license}
                onChange={(e) => setForm((f) => ({ ...f, license: e.target.value }))}
              />
            </div>
            <div className="form-group">
              <label htmlFor="skill-compat">Compatibility (optional, max 500 chars)</label>
              <input
                id="skill-compat"
                type="text"
                className="input"
                value={form.compatibility}
                onChange={(e) => setForm((f) => ({ ...f, compatibility: e.target.value }))}
                maxLength={500}
              />
            </div>
            <div className="form-group">
              <label htmlFor="skill-allowed-tools">Allowed tools (optional)</label>
              <input
                id="skill-allowed-tools"
                type="text"
                className="input"
                value={form.allowedTools}
                onChange={(e) => setForm((f) => ({ ...f, allowedTools: e.target.value }))}
                placeholder="tool1, tool2"
              />
            </div>

            <h3 className="section-title">Content</h3>
            <div className="form-group">
              <label htmlFor="skill-content">Skill content (markdown)</label>
              <textarea
                id="skill-content"
                className="input"
                value={form.content}
                onChange={(e) => setForm((f) => ({ ...f, content: e.target.value }))}
                rows={12}
                style={{ fontFamily: 'monospace', fontSize: '0.9rem' }}
              />
            </div>

            <div className="form-actions" style={{ display: 'flex', gap: '0.5rem', marginTop: '1.5rem', paddingTop: '1rem', borderTop: '1px solid var(--color-border)' }}>
              <button type="submit" className="btn btn-primary" disabled={saving}>
                {saving ? 'Saving...' : (isNew ? 'Create skill' : 'Save changes')}
              </button>
              <Link to="/skills" className="btn btn-secondary">Cancel</Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

export default SkillEdit;
