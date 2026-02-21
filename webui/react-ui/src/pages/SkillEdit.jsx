import { useState, useEffect } from 'react';
import { useParams, useNavigate, useLocation, Link, useOutletContext } from 'react-router-dom';
import { skillsApi } from '../utils/api';

const RESOURCE_PREFIXES = ['scripts/', 'references/', 'assets/'];
function isValidResourcePath(path) {
  return RESOURCE_PREFIXES.some((p) => path.startsWith(p)) && !path.includes('..');
}

function ResourceGroup({ title, icon, items, readOnly, pathPrefix, onView, onDelete, onUpload }) {
  return (
    <div className="resource-section" style={{ marginBottom: '1.5rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
        <h3 className="section-title" style={{ marginBottom: 0 }}>
          <i className={`fas fa-${icon}`} /> {title}
        </h3>
        {!readOnly && (
          <button type="button" className="action-btn success" onClick={() => onUpload(pathPrefix)}>
            <i className="fas fa-upload" /> Upload
          </button>
        )}
      </div>
      <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
        {items.length === 0 ? (
          <li style={{ color: 'var(--color-text-secondary)', padding: '0.75rem', fontSize: '0.9rem' }}>No {title.toLowerCase()} yet.</li>
        ) : (
          items.map((res) => (
            <li key={res.path} className="card" style={{ padding: '0.5rem 0.75rem', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '0.5rem' }}>
              <div style={{ minWidth: 0 }}>
                <span style={{ fontWeight: 500 }}>{res.name}</span>
                <span style={{ color: 'var(--color-text-secondary)', fontSize: '0.85rem', marginLeft: '0.5rem' }}>{res.mime_type} · {(res.size || 0).toLocaleString()} B</span>
              </div>
              <div style={{ display: 'flex', gap: '0.5rem' }}>
                <button type="button" className="action-btn" onClick={() => onView(res)}>
                  <i className="fas fa-edit" /> View/Edit
                </button>
                {!readOnly && (
                  <button type="button" className="action-btn delete-btn" onClick={() => onDelete(res.path)}>
                    <i className="fas fa-trash" /> Delete
                  </button>
                )}
              </div>
            </li>
          ))
        )}
      </ul>
    </div>
  );
}

function ResourcesSection({ skillName, showToast }) {
  const [data, setData] = useState({ scripts: [], references: [], assets: [], readOnly: false });
  const [loading, setLoading] = useState(true);
  const [editor, setEditor] = useState({ open: false, path: '', name: '', content: '', readable: true, saving: false });
  const [upload, setUpload] = useState({ open: false, pathPrefix: 'assets/', file: null, pathInput: '', uploading: false });
  const [deletePath, setDeletePath] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const res = await skillsApi.listResources(skillName);
      setData({
        scripts: res.scripts || [],
        references: res.references || [],
        assets: res.assets || [],
        readOnly: res.readOnly === true,
      });
    } catch (err) {
      showToast(err.message || 'Failed to load resources', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [skillName]);

  const handleView = async (res) => {
    setEditor({ open: true, path: res.path, name: res.name, content: '', readable: res.readable !== false, saving: false });
    if (res.readable !== false) {
      try {
        const json = await skillsApi.getResource(skillName, res.path, { json: true });
        const content = json.encoding === 'base64' && json.content ? atob(json.content) : (json.content || '');
        setEditor((e) => ({ ...e, content }));
      } catch (err) {
        showToast(err.message || 'Failed to load file', 'error');
      }
    }
  };

  const handleEditorSave = async () => {
    setEditor((e) => ({ ...e, saving: true }));
    try {
      await skillsApi.updateResource(skillName, editor.path, editor.content);
      showToast('Resource updated', 'success');
      setEditor((e) => ({ ...e, open: false }));
      load();
    } catch (err) {
      showToast(err.message || 'Update failed', 'error');
    } finally {
      setEditor((e) => ({ ...e, saving: false }));
    }
  };

  const handleUploadOpen = (pathPrefix) => {
    setUpload({ open: true, pathPrefix, file: null, pathInput: '', uploading: false });
  };

  const handleUploadSubmit = async () => {
    const path = upload.pathInput.trim() || (upload.file ? upload.pathPrefix + upload.file.name : '');
    if (!path || !upload.file) {
      showToast('Select a file and ensure path is set', 'error');
      return;
    }
    if (!isValidResourcePath(path)) {
      showToast('Path must start with scripts/, references/, or assets/', 'error');
      return;
    }
    setUpload((u) => ({ ...u, uploading: true }));
    try {
      await skillsApi.createResource(skillName, path, upload.file);
      showToast('Resource added', 'success');
      setUpload((u) => ({ ...u, open: false }));
      load();
    } catch (err) {
      showToast(err.message || 'Upload failed', 'error');
    } finally {
      setUpload((u) => ({ ...u, uploading: false }));
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deletePath) return;
    try {
      await skillsApi.deleteResource(skillName, deletePath);
      showToast('Resource deleted', 'success');
      setDeletePath(null);
      load();
    } catch (err) {
      showToast(err.message || 'Delete failed', 'error');
    }
  };

  return (
    <>
      <div className="section-box">
        <h2><i className="fas fa-folder" /> Resources</h2>
        <p className="page-description" style={{ marginBottom: '1rem' }}>Scripts, references, and assets for this skill. Paths must start with scripts/, references/, or assets/.</p>
        {loading ? (
          <p>Loading resources...</p>
        ) : (
          <>
            <ResourceGroup title="Scripts" icon="code" pathPrefix="scripts/" items={data.scripts} readOnly={data.readOnly} onView={handleView} onDelete={setDeletePath} onUpload={handleUploadOpen} />
            <ResourceGroup title="References" icon="book" pathPrefix="references/" items={data.references} readOnly={data.readOnly} onView={handleView} onDelete={setDeletePath} onUpload={handleUploadOpen} />
            <ResourceGroup title="Assets" icon="image" pathPrefix="assets/" items={data.assets} readOnly={data.readOnly} onView={handleView} onDelete={setDeletePath} onUpload={handleUploadOpen} />
          </>
        )}
      </div>

      {editor.open && (
        <div className="modal-overlay" style={{ position: 'fixed', inset: 0, background: 'var(--color-bg-overlay)', zIndex: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '1rem' }} onClick={() => !editor.saving && setEditor((e) => ({ ...e, open: false }))}>
          <div className="card" style={{ maxWidth: '700px', width: '100%', maxHeight: '90vh', display: 'flex', flexDirection: 'column' }} onClick={(e) => e.stopPropagation()}>
            <h3 className="section-title" style={{ marginTop: 0 }}>Edit {editor.name}</h3>
            {editor.readable ? (
              <>
                <textarea className="input" value={editor.content} onChange={(e) => setEditor((x) => ({ ...x, content: e.target.value }))} rows={14} style={{ flex: 1, fontFamily: 'monospace', fontSize: '0.9rem', marginBottom: '1rem' }} />
                <div className="form-actions" style={{ display: 'flex', gap: '1rem', justifyContent: 'flex-end' }}>
                  <button type="button" className="action-btn" onClick={() => setEditor((e) => ({ ...e, open: false }))}>Cancel</button>
                  <button type="button" className="action-btn success" disabled={editor.saving} onClick={handleEditorSave}>{editor.saving ? 'Saving...' : 'Save'}</button>
                </div>
              </>
            ) : (
              <p style={{ color: 'var(--color-text-secondary)' }}>Binary file. Download via API or export skill.</p>
            )}
          </div>
        </div>
      )}

      {upload.open && (
        <div className="modal-overlay" style={{ position: 'fixed', inset: 0, background: 'var(--color-bg-overlay)', zIndex: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '1rem' }} onClick={() => !upload.uploading && setUpload((u) => ({ ...u, open: false }))}>
          <div className="card" style={{ maxWidth: '400px', width: '100%' }} onClick={(e) => e.stopPropagation()}>
            <h3 className="section-title" style={{ marginTop: 0 }}>Upload to {upload.pathPrefix}</h3>
            <div className="form-group">
              <label>File</label>
              <input type="file" className="input" onChange={(e) => setUpload((u) => ({ ...u, file: e.target.files?.[0] || null }))} />
            </div>
            <div className="form-group">
              <label>Path (default: {upload.pathPrefix} + filename)</label>
              <input type="text" className="input" placeholder={`${upload.pathPrefix}filename`} value={upload.pathInput} onChange={(e) => setUpload((u) => ({ ...u, pathInput: e.target.value }))} />
            </div>
            <div className="form-actions" style={{ display: 'flex', gap: '1rem', justifyContent: 'flex-end' }}>
              <button type="button" className="action-btn" onClick={() => setUpload((u) => ({ ...u, open: false }))}>Cancel</button>
              <button type="button" className="action-btn success" disabled={upload.uploading || !upload.file} onClick={handleUploadSubmit}>{upload.uploading ? 'Uploading...' : 'Upload'}</button>
            </div>
          </div>
        </div>
      )}

      {deletePath && (
        <div className="modal-overlay" style={{ position: 'fixed', inset: 0, background: 'var(--color-bg-overlay)', zIndex: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '1rem' }} onClick={() => setDeletePath(null)}>
          <div className="card" style={{ maxWidth: '360px' }} onClick={(e) => e.stopPropagation()}>
            <p>Delete resource <strong>{deletePath}</strong>?</p>
            <div className="form-actions" style={{ display: 'flex', gap: '1rem', justifyContent: 'flex-end' }}>
              <button type="button" className="action-btn" onClick={() => setDeletePath(null)}>Cancel</button>
              <button type="button" className="action-btn delete-btn" onClick={handleDeleteConfirm}>Delete</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function SkillEdit() {
  const { name: nameParam } = useParams();
  const location = useLocation();
  const isNew = location.pathname.endsWith('/new');
  const name = nameParam ? decodeURIComponent(nameParam) : undefined;
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [activeSection, setActiveSection] = useState('basic-section');
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
        <div>
          <Link to="/skills" className="back-link" style={{ marginBottom: '0.5rem', display: 'inline-block' }}>
            <i className="fas fa-arrow-left" /> Back to skills
          </Link>
          <h1>
            <i className="fas fa-book" /> {isNew ? 'New skill' : `Edit: ${name}`}
          </h1>
        </div>
      </header>

      <div className="create-agent-content">
        <div className="section-box">
          <h2>
            <i className="fas fa-cog" /> Skill configuration
          </h2>

          <div className="agent-form-container">
            <div className="wizard-sidebar">
              <ul className="wizard-nav">
                <li
                  className={`wizard-nav-item ${activeSection === 'basic-section' ? 'active' : ''}`}
                  onClick={() => setActiveSection('basic-section')}
                >
                  <i className="fas fa-info-circle" /> Basic information
                </li>
                <li
                  className={`wizard-nav-item ${activeSection === 'content-section' ? 'active' : ''}`}
                  onClick={() => setActiveSection('content-section')}
                >
                  <i className="fas fa-file-alt" /> Content
                </li>
                <li
                  className={`wizard-nav-item ${activeSection === 'resources-section' ? 'active' : ''}`}
                  onClick={() => setActiveSection('resources-section')}
                >
                  <i className="fas fa-folder" /> Resources
                </li>
              </ul>
            </div>

            <div className="form-content-area">
              <form className="agent-form" onSubmit={handleSubmit} noValidate>
                <div style={{ display: activeSection === 'basic-section' ? 'block' : 'none' }}>
                  <h3 className="section-title">Basic information</h3>
                  <div className="mb-4">
                    <label htmlFor="skill-name">Name (lowercase, hyphens only) <span style={{ color: 'var(--color-error)' }}>*</span></label>
                    <input
                      id="skill-name"
                      name="name"
                      type="text"
                      className="input"
                      value={form.name}
                      onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                      required
                      disabled={!isNew}
                      placeholder="my-skill"
                    />
                    {!isNew && <p className="help-text" style={{ marginTop: '0.5rem', fontSize: '0.9rem', color: 'var(--color-text-secondary)' }}>Name cannot be changed after creation.</p>}
                  </div>
                  <div className="mb-4">
                    <label htmlFor="skill-desc">Description (required, 1–1024 chars) <span style={{ color: 'var(--color-error)' }}>*</span></label>
                    <textarea
                      id="skill-desc"
                      name="description"
                      className="input"
                      value={form.description}
                      onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                      required
                      maxLength={1024}
                      rows={2}
                    />
                  </div>
                  <div className="mb-4">
                    <label htmlFor="skill-license">License (optional)</label>
                    <input
                      id="skill-license"
                      name="license"
                      type="text"
                      className="input"
                      value={form.license}
                      onChange={(e) => setForm((f) => ({ ...f, license: e.target.value }))}
                    />
                  </div>
                  <div className="mb-4">
                    <label htmlFor="skill-compat">Compatibility (optional, max 500 chars)</label>
                    <input
                      id="skill-compat"
                      name="compatibility"
                      type="text"
                      className="input"
                      value={form.compatibility}
                      onChange={(e) => setForm((f) => ({ ...f, compatibility: e.target.value }))}
                      maxLength={500}
                    />
                  </div>
                  <div className="mb-4">
                    <label htmlFor="skill-allowed-tools">Allowed tools (optional)</label>
                    <input
                      id="skill-allowed-tools"
                      name="allowedTools"
                      type="text"
                      className="input"
                      value={form.allowedTools}
                      onChange={(e) => setForm((f) => ({ ...f, allowedTools: e.target.value }))}
                      placeholder="tool1, tool2"
                    />
                  </div>
                </div>

                <div style={{ display: activeSection === 'content-section' ? 'block' : 'none' }}>
                  <h3 className="section-title">Content</h3>
                  <div className="mb-4">
                    <label htmlFor="skill-content">Skill content (markdown)</label>
                    <textarea
                      id="skill-content"
                      name="content"
                      className="input"
                      value={form.content}
                      onChange={(e) => setForm((f) => ({ ...f, content: e.target.value }))}
                      rows={14}
                      style={{ fontFamily: 'monospace', fontSize: '0.9rem' }}
                    />
                  </div>
                </div>

                {activeSection === 'resources-section' && (
                  <div style={{ display: 'block' }}>
                    {isNew || !name ? (
                      <div className="section-box" style={{ padding: '1.5rem', marginTop: 0 }}>
                        <h3 className="section-title">Resources</h3>
                        <p style={{ color: 'var(--color-text-secondary)', marginBottom: 0 }}>
                          Save the skill first to add scripts, references, and assets. After creating the skill, use this tab to upload files and manage resources.
                        </p>
                      </div>
                    ) : (
                      <ResourcesSection skillName={name} showToast={showToast} />
                    )}
                  </div>
                )}

                <div className="form-actions" style={{ display: 'flex', gap: '1rem', justifyContent: 'flex-end', marginTop: '1.5rem', paddingTop: '1rem', borderTop: '1px solid var(--color-border)' }}>
                  <Link to="/skills" className="action-btn">
                    <i className="fas fa-times" /> Cancel
                  </Link>
                  <button type="submit" className="action-btn success" disabled={saving}>
                    <i className="fas fa-save" /> {saving ? 'Saving...' : (isNew ? 'Create skill' : 'Save changes')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default SkillEdit;
