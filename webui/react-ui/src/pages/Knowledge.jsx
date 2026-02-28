import { useState, useEffect } from 'react';
import { useOutletContext } from 'react-router-dom';
import { collectionsApi } from '../utils/api';

const TABS = [
  { id: 'search', label: 'Search', icon: 'fa-search' },
  { id: 'collections', label: 'Collections', icon: 'fa-folder' },
  { id: 'upload', label: 'Upload', icon: 'fa-upload' },
  { id: 'sources', label: 'Sources', icon: 'fa-globe' },
  { id: 'entries', label: 'Entries', icon: 'fa-list' },
];

function Knowledge() {
  const { showToast } = useOutletContext();
  const [tab, setTab] = useState('search');
  const [collections, setCollections] = useState([]);
  const [loadingCollections, setLoadingCollections] = useState(true);

  const fetchCollections = async () => {
    setLoadingCollections(true);
    try {
      const list = await collectionsApi.list();
      setCollections(Array.isArray(list) ? list : []);
    } catch (err) {
      showToast(err.message || 'Failed to load collections', 'error');
      setCollections([]);
    } finally {
      setLoadingCollections(false);
    }
  };

  useEffect(() => {
    document.title = 'Knowledge base - LocalAGI';
    fetchCollections();
  }, []);

  return (
    <div className="page knowledge-page">
      <header className="page-header">
        <h1 className="page-title">
          <i className="fas fa-database" />
          Knowledge base
        </h1>
        <p className="page-description">
          Manage collections, upload files, search content, and sync external sources.
        </p>
      </header>

      <div className="knowledge-tabs">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            className={`tab-btn ${tab === t.id ? 'active' : ''}`}
            onClick={() => setTab(t.id)}
          >
            <i className={`fas ${t.icon}`} />
            <span>{t.label}</span>
          </button>
        ))}
      </div>

      <div className="knowledge-content">
        {tab === 'search' && (
          <SearchTab
            collections={collections}
            loadingCollections={loadingCollections}
            onRefreshCollections={fetchCollections}
            showToast={showToast}
          />
        )}
        {tab === 'collections' && (
          <CollectionsTab
            collections={collections}
            loadingCollections={loadingCollections}
            onRefresh={fetchCollections}
            showToast={showToast}
          />
        )}
        {tab === 'upload' && (
          <UploadTab
            collections={collections}
            loadingCollections={loadingCollections}
            onRefreshCollections={fetchCollections}
            showToast={showToast}
          />
        )}
        {tab === 'sources' && (
          <SourcesTab
            collections={collections}
            loadingCollections={loadingCollections}
            showToast={showToast}
          />
        )}
        {tab === 'entries' && (
          <EntriesTab
            collections={collections}
            loadingCollections={loadingCollections}
            onRefreshCollections={fetchCollections}
            showToast={showToast}
          />
        )}
      </div>
    </div>
  );
}

function SearchTab({ collections, loadingCollections, onRefreshCollections, showToast }) {
  const [selectedCollection, setSelectedCollection] = useState('');
  const [query, setQuery] = useState('');
  const [maxResults, setMaxResults] = useState(5);
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSearch = async () => {
    if (!selectedCollection || !query.trim()) {
      setError('Select a collection and enter a query');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const list = await collectionsApi.search(selectedCollection, query.trim(), maxResults || 5);
      setResults(Array.isArray(list) ? list : []);
      if ((list?.length ?? 0) === 0) {
        setResults([{ Content: `No results for "${query}"` }]);
      }
    } catch (err) {
      showToast(err.message || 'Search failed', 'error');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="knowledge-card">
      <h2 className="knowledge-card-title">Search collections</h2>
      <p className="knowledge-card-desc">Semantic search over your indexed content.</p>
      {error && <div className="knowledge-error">{error}</div>}
      <div className="form-group">
        <label>Collection</label>
        <select
          value={selectedCollection}
          onChange={(e) => setSelectedCollection(e.target.value)}
          disabled={loadingCollections}
        >
          <option value="">Select a collection</option>
          {collections.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>
      <div className="form-group">
        <label>Query</label>
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          placeholder="Enter search query..."
        />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Max results</label>
          <input
            type="number"
            min={1}
            max={20}
            value={maxResults}
            onChange={(e) => setMaxResults(Number(e.target.value) || 5)}
          />
        </div>
        <button type="button" className="btn btn-primary" onClick={handleSearch} disabled={loading}>
          {loading ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-search" />}
          <span>{loading ? 'Searching...' : 'Search'}</span>
        </button>
      </div>
      <div className="search-results">
        <h3>Results</h3>
        {results.length > 0 ? (
          <ul className="results-list">
            {results.map((r, i) => (
              <li key={i} className="result-item">
                <pre>{typeof r === 'object' && r.Content != null ? r.Content : JSON.stringify(r, null, 2)}</pre>
              </li>
            ))}
          </ul>
        ) : (
          !loading && <p className="muted">Run a search to see results.</p>
        )}
      </div>
    </section>
  );
}

function CollectionsTab({ collections, loadingCollections, onRefresh, showToast }) {
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);
  const [resetting, setResetting] = useState(null);

  const handleCreate = async () => {
    if (!newName.trim()) {
      showToast('Enter a collection name', 'error');
      return;
    }
    setCreating(true);
    try {
      await collectionsApi.create(newName.trim());
      showToast(`Collection "${newName}" created`, 'success');
      setNewName('');
      onRefresh();
    } catch (err) {
      showToast(err.message || 'Failed to create collection', 'error');
    } finally {
      setCreating(false);
    }
  };

  const handleReset = async (name) => {
    if (!confirm(`Reset collection "${name}"? This removes all entries and cannot be undone.`)) return;
    setResetting(name);
    try {
      await collectionsApi.reset(name);
      showToast(`Collection "${name}" reset`, 'success');
      onRefresh();
    } catch (err) {
      showToast(err.message || 'Failed to reset', 'error');
    } finally {
      setResetting(null);
    }
  };

  return (
    <section className="knowledge-card">
      <h2 className="knowledge-card-title">Create collection</h2>
      <div className="form-row">
        <input
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
          placeholder="Collection name..."
          className="flex-1"
        />
        <button type="button" className="btn btn-primary" onClick={handleCreate} disabled={creating}>
          {creating ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-plus" />}
          <span>{creating ? 'Creating...' : 'Create'}</span>
        </button>
      </div>
      <div className="form-row" style={{ alignItems: 'center', marginBottom: '0.75rem' }}>
        <h2 className="knowledge-card-title" style={{ margin: 0 }}>Your collections</h2>
        <button type="button" className="btn btn-ghost icon-only" onClick={onRefresh} disabled={loadingCollections} title="Refresh">
          <i className={loadingCollections ? 'fas fa-spinner fa-spin' : 'fas fa-sync-alt'} />
        </button>
      </div>
      {loadingCollections ? (
        <p className="muted">Loading...</p>
      ) : collections.length === 0 ? (
        <p className="muted">No collections. Create one above.</p>
      ) : (
        <ul className="knowledge-list">
          {collections.map((c) => (
            <li key={c} className="knowledge-list-item">
              <i className="fas fa-folder" />
              <span>{c}</span>
              <button
                type="button"
                className="btn btn-ghost danger"
                onClick={() => handleReset(c)}
                disabled={resetting === c}
                title="Reset collection"
              >
                {resetting === c ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-redo-alt" />}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function UploadTab({ collections, loadingCollections, onRefreshCollections, showToast }) {
  const [selectedCollection, setSelectedCollection] = useState('');
  const [file, setFile] = useState(null);
  const [uploading, setUploading] = useState(false);

  const handleUpload = async () => {
    if (!selectedCollection) {
      showToast('Select a collection', 'error');
      return;
    }
    if (!file) {
      showToast('Select a file', 'error');
      return;
    }
    setUploading(true);
    try {
      await collectionsApi.upload(selectedCollection, file);
      showToast('File uploaded', 'success');
      setFile(null);
      onRefreshCollections();
    } catch (err) {
      showToast(err.message || 'Upload failed', 'error');
    } finally {
      setUploading(false);
    }
  };

  return (
    <section className="knowledge-card">
      <h2 className="knowledge-card-title">Upload file</h2>
      <div className="form-group">
        <label>Collection</label>
        <select
          value={selectedCollection}
          onChange={(e) => setSelectedCollection(e.target.value)}
          disabled={loadingCollections}
        >
          <option value="">Select a collection</option>
          {collections.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>
      <div className="form-group">
        <label>File</label>
        <input
          type="file"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
        />
        {file && <span className="muted">{file.name}</span>}
      </div>
      <button type="button" className="btn btn-primary" onClick={handleUpload} disabled={uploading}>
        {uploading ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-upload" />}
        <span>{uploading ? 'Uploading...' : 'Upload'}</span>
      </button>
    </section>
  );
}

function SourcesTab({ collections, loadingCollections, showToast }) {
  const [selectedCollection, setSelectedCollection] = useState('');
  const [url, setUrl] = useState('');
  const [intervalMin, setIntervalMin] = useState(60);
  const [sources, setSources] = useState([]);
  const [loadingSources, setLoadingSources] = useState(false);
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState(null);

  useEffect(() => {
    if (!selectedCollection) {
      setSources([]);
      return;
    }
    let cancelled = false;
    setLoadingSources(true);
    collectionsApi.listSources(selectedCollection)
      .then((list) => { if (!cancelled) setSources(Array.isArray(list) ? list : []); })
      .catch((err) => { if (!cancelled) showToast(err.message || 'Failed to load sources', 'error'); })
      .finally(() => { if (!cancelled) setLoadingSources(false); });
    return () => { cancelled = true; };
  }, [selectedCollection, showToast]);

  const handleAdd = async () => {
    if (!selectedCollection || !url.trim()) {
      showToast('Select a collection and enter a URL', 'error');
      return;
    }
    setAdding(true);
    try {
      await collectionsApi.addSource(selectedCollection, url.trim(), intervalMin || 60);
      showToast('Source added', 'success');
      setUrl('');
      setSources(await collectionsApi.listSources(selectedCollection));
    } catch (err) {
      showToast(err.message || 'Failed to add source', 'error');
    } finally {
      setAdding(false);
    }
  };

  const handleRemove = async (sourceUrl) => {
    if (!selectedCollection) return;
    setRemoving(sourceUrl);
    try {
      await collectionsApi.removeSource(selectedCollection, sourceUrl);
      showToast('Source removed', 'success');
      setSources(await collectionsApi.listSources(selectedCollection));
    } catch (err) {
      showToast(err.message || 'Failed to remove source', 'error');
    } finally {
      setRemoving(null);
    }
  };

  return (
    <section className="knowledge-card">
      <h2 className="knowledge-card-title">External sources</h2>
      <p className="knowledge-card-desc">Sync URLs to a collection periodically.</p>
      <div className="form-group">
        <label>Collection</label>
        <select
          value={selectedCollection}
          onChange={(e) => setSelectedCollection(e.target.value)}
          disabled={loadingCollections}
        >
          <option value="">Select a collection</option>
          {collections.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>
      <div className="form-row">
        <div className="form-group flex-1">
          <label>URL</label>
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com"
          />
        </div>
        <div className="form-group">
          <label>Interval (min)</label>
          <input
            type="number"
            min={1}
            value={intervalMin}
            onChange={(e) => setIntervalMin(Number(e.target.value) || 60)}
          />
        </div>
      </div>
      <button type="button" className="btn btn-primary" onClick={handleAdd} disabled={adding}>
        {adding ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-plus" />}
        <span>{adding ? 'Adding...' : 'Add source'}</span>
      </button>
      <h3 className="knowledge-card-title">Registered sources</h3>
      {loadingSources ? (
        <p className="muted">Loading...</p>
      ) : sources.length === 0 ? (
        <p className="muted">No sources. Add one above.</p>
      ) : (
        <ul className="knowledge-list">
          {sources.map((s) => (
            <li key={s.url} className="knowledge-list-item">
              <i className="fas fa-globe" />
              <div>
                <span>{s.url}</span>
                <span className="muted">Every {s.update_interval ?? 60} min</span>
              </div>
              <button
                type="button"
                className="btn btn-ghost danger"
                onClick={() => handleRemove(s.url)}
                disabled={removing === s.url}
                title="Remove"
              >
                {removing === s.url ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-trash" />}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function EntriesTab({ collections, loadingCollections, onRefreshCollections, showToast }) {
  const [selectedCollection, setSelectedCollection] = useState('');
  const [entries, setEntries] = useState([]);
  const [loadingEntries, setLoadingEntries] = useState(false);
  const [deleting, setDeleting] = useState(null);
  const [resetting, setResetting] = useState(null);
  const [viewContent, setViewContent] = useState(null);
  const [loadingContent, setLoadingContent] = useState(null);

  useEffect(() => {
    if (!selectedCollection) {
      setEntries([]);
      return;
    }
    let cancelled = false;
    setLoadingEntries(true);
    collectionsApi.listEntries(selectedCollection)
      .then((list) => { if (!cancelled) setEntries(Array.isArray(list) ? list : []); })
      .catch((err) => { if (!cancelled) showToast(err.message || 'Failed to load entries', 'error'); })
      .finally(() => { if (!cancelled) setLoadingEntries(false); });
    return () => { cancelled = true; };
  }, [selectedCollection, showToast]);

  const handleDelete = async (entry) => {
    if (!selectedCollection) return;
    if (!confirm(`Delete "${entry}" from collection?`)) return;
    setDeleting(entry);
    try {
      await collectionsApi.deleteEntry(selectedCollection, entry);
      showToast('Entry deleted', 'success');
      setEntries(await collectionsApi.listEntries(selectedCollection));
    } catch (err) {
      showToast(err.message || 'Failed to delete', 'error');
    } finally {
      setDeleting(null);
    }
  };

  const handleReset = async () => {
    if (!selectedCollection) return;
    if (!confirm(`Reset collection "${selectedCollection}"? This removes all entries.`)) return;
    setResetting(selectedCollection);
    try {
      await collectionsApi.reset(selectedCollection);
      showToast('Collection reset', 'success');
      setEntries([]);
      onRefreshCollections();
    } catch (err) {
      showToast(err.message || 'Failed to reset', 'error');
    } finally {
      setResetting(null);
    }
  };

  const handleViewContent = async (entry) => {
    if (!selectedCollection) return;
    setLoadingContent(entry);
    try {
      const { content, chunkCount } = await collectionsApi.getEntryContent(selectedCollection, entry);
      setViewContent({ entry, content, chunkCount });
    } catch (err) {
      showToast(err.message || 'Failed to load content', 'error');
    } finally {
      setLoadingContent(null);
    }
  };

  return (
    <section className="knowledge-card">
      <h2 className="knowledge-card-title">Collection entries</h2>
      <div className="form-group">
        <label>Collection</label>
        <select
          value={selectedCollection}
          onChange={(e) => setSelectedCollection(e.target.value)}
          disabled={loadingCollections}
        >
          <option value="">Select a collection</option>
          {collections.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>
      <div className="form-row">
        <button
          type="button"
          className="btn btn-ghost danger"
          onClick={handleReset}
          disabled={!selectedCollection || resetting === selectedCollection}
        >
          {resetting === selectedCollection ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-redo-alt" />}
          <span>Reset collection</span>
        </button>
      </div>
      {loadingEntries ? (
        <p className="muted">Loading entries...</p>
      ) : entries.length === 0 ? (
        <p className="muted">Select a collection to view entries.</p>
      ) : (
        <ul className="knowledge-list">
          {entries.map((entry) => (
            <li key={entry} className="knowledge-list-item">
              <i className="fas fa-file-alt" />
              <span className="truncate">{entry}</span>
              <div className="btn-group">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => handleViewContent(entry)}
                  disabled={loadingContent === entry}
                  title="View content"
                >
                  {loadingContent === entry ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-eye" />}
                </button>
                <button
                  type="button"
                  className="btn btn-ghost danger"
                  onClick={() => handleDelete(entry)}
                  disabled={deleting === entry}
                  title="Delete"
                >
                  {deleting === entry ? <i className="fas fa-spinner fa-spin" /> : <i className="fas fa-trash" />}
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
      {viewContent && (
        <div className="modal-overlay" onClick={() => setViewContent(null)}>
          <div className="modal-content knowledge-modal" onClick={(e) => e.stopPropagation()}>
            <h3>Content: {viewContent.entry}</h3>
            <p className="muted">{viewContent.chunkCount} chunk(s)</p>
            <pre className="knowledge-entry-content">{viewContent.content || '(empty)'}</pre>
            <button type="button" className="btn btn-primary" onClick={() => setViewContent(null)}>Close</button>
          </div>
        </div>
      )}
    </section>
  );
}

export default Knowledge;
