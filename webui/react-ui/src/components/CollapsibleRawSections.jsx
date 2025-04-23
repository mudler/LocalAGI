import React, { useState } from 'react';
import hljs from 'highlight.js/lib/core';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/monokai.css';

hljs.registerLanguage('json', json);

export default function CollapsibleRawSections({ container }) {
  const [showCreation, setShowCreation] = useState(false);
  const [showProgress, setShowProgress] = useState(false);
  const [showCompletion, setShowCompletion] = useState(false);
  const [copied, setCopied] = useState({ creation: false, progress: false, completion: false });

  const handleCopy = (section, data) => {
    navigator.clipboard.writeText(JSON.stringify(data, null, 2));
    setCopied(prev => ({ ...prev, [section]: true }));
    setTimeout(() => setCopied(prev => ({ ...prev, [section]: false })), 1200);
  };

  return (
    <div>
      {/* Creation Section */}
      {container.creation && (
        <div>
          <h4 style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', flex: 1 }}
              onClick={() => setShowCreation(v => !v)}
            >
              <i className={`fas fa-chevron-${showCreation ? 'down' : 'right'}`} style={{ marginRight: 6 }} />
              Creation
            </span>
            <button
              title="Copy Creation JSON"
              onClick={e => { e.stopPropagation(); handleCopy('creation', container.creation); }}
              style={{ marginLeft: 8, border: 'none', background: 'none', cursor: 'pointer', color: '#ccc' }}
            >
              {copied.creation ? <span style={{ color: '#6f6' }}>Copied!</span> : <i className="fas fa-copy" />}
            </button>
          </h4>
          {showCreation && (
            <pre className="hljs"><code>
              <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(container.creation || {}, null, 2), { language: 'json' }).value }} />
            </code></pre>
          )}
        </div>
      )}
      {/* Progress Section */}
      {container.progress && container.progress.length > 0 && (
        <div>
          <h4 style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', flex: 1 }}
              onClick={() => setShowProgress(v => !v)}
            >
              <i className={`fas fa-chevron-${showProgress ? 'down' : 'right'}`} style={{ marginRight: 6 }} />
              Progress
            </span>
            <button
              title="Copy Progress JSON"
              onClick={e => { e.stopPropagation(); handleCopy('progress', container.progress); }}
              style={{ marginLeft: 8, border: 'none', background: 'none', cursor: 'pointer', color: '#ccc' }}
            >
              {copied.progress ? <span style={{ color: '#6f6' }}>Copied!</span> : <i className="fas fa-copy" />}
            </button>
          </h4>
          {showProgress && (
            <pre className="hljs"><code>
              <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(container.progress || {}, null, 2), { language: 'json' }).value }} />
            </code></pre>
          )}
        </div>
      )}
      {/* Completion Section */}
      {container.completion && (
        <div>
          <h4 style={{ display: 'flex', alignItems: 'center' }}>
            <span
              style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', flex: 1 }}
              onClick={() => setShowCompletion(v => !v)}
            >
              <i className={`fas fa-chevron-${showCompletion ? 'down' : 'right'}`} style={{ marginRight: 6 }} />
              Completion
            </span>
            <button
              title="Copy Completion JSON"
              onClick={e => { e.stopPropagation(); handleCopy('completion', container.completion); }}
              style={{ marginLeft: 8, border: 'none', background: 'none', cursor: 'pointer', color: '#ccc' }}
            >
              {copied.completion ? <span style={{ color: '#6f6' }}>Copied!</span> : <i className="fas fa-copy" />}
            </button>
          </h4>
          {showCompletion && (
            <pre className="hljs"><code>
              <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(container.completion || {}, null, 2), { language: 'json' }).value }} />
            </code></pre>
          )}
        </div>
      )}
    </div>
  );
}

