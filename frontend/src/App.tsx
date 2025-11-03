import React, { useMemo, useRef, useState } from "react";
import { Search, ExternalLink, SlidersHorizontal, Loader2, LogIn, LogOut } from "lucide-react";
import { toGitHubUrl } from "./github";

// If your UI runs on a different origin/port than the API, set VITE_API_BASE
//   echo 'VITE_API_BASE=http://localhost:8080' > .env.local
const API_BASE = (import.meta as any)?.env?.VITE_API_BASE || ""; // same-origin by default

interface GitHubUser {
  login: string;
  name: string;
  avatar_url: string;
  email?: string;
}

// Matches /search?format=simple
interface SimpleResult {
  path: string;
  language: string;
  line_start: number;
  line_end: number;
  score: number;
  preview?: string;
  summary?: string;
  ref?: string; // The ref/branch this result came from
  repository?: string; // The repository this result came from
}

// Minimal CSS (no Tailwind required)
const styles = `
:root{color-scheme:light dark;}
*{box-sizing:border-box}
html,body{background:#0b0b0b}
body{margin:0;display:flex;justify-content:center;align-items:flex-start;font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Inter,Arial,Apple Color Emoji,Segoe UI Emoji;color:#eee}
.app{width:100%;max-width:900px;padding:40px 20px}
.header { background: #0b0b0b; border-bottom: 1px solid #191919; position: sticky; top: 0; }
.title{font-weight:800;font-size:40px;margin:6px 0 4px;text-align:center}
.subtitle{font-size:15px;color:#aaa;text-align:center}
.api{font-size:12px;color:#666;text-align:center;margin-bottom:20px}
.card{border:1px solid #222;border-radius:18px;padding:20px;background:#161616;margin:0 auto}
.row{display:flex;gap:12px;align-items:center;justify-content:center}
.input{flex:1;display:flex;align-items:center;gap:10px;border:1px solid #333;border-radius:16px;padding:16px 18px;background:#0f0f0f;max-width:700px}
.input input{flex:1;background:transparent;border:none;outline:none;color:#eee;font-size:18px;text-align:left}
.btn{display:inline-flex;align-items:center;gap:10px;border:1px solid #333;border-radius:14px;padding:14px 18px;background:#1c1c1c;color:#eee;font-weight:700;cursor:pointer}
.btn[disabled]{opacity:.6;cursor:not-allowed}
.filters summary{cursor:pointer;color:#bbb;display:inline-flex;align-items:center;gap:8px;justify-content:flex-start;width:100%}
.filters .grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px;margin-top:10px}
/* Ensure label rows inside filters align left and keep label width */
.filters .grid .row{display:flex;align-items:center;justify-content:flex-start;gap:8px}
.filters .grid .row > span{flex:0 0 110px}
/* Ensure inputs/selects can shrink inside grid cells and not overflow labels */
.filters .grid .row > .select, .filters .grid .row > .text, .filters .grid .row > select, .filters .grid .row > input{flex:1;min-width:0}
.select,.text{border:1px solid #333;border-radius:10px;padding:8px 10px;background:#0f0f0f;color:#eee;height:36px;min-height:36px;max-width:100%}
.select{appearance:none;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
/* Allow native dropdown to overflow visually when focused */
.select:focus{position:relative;z-index:9999}
.checkbox{accent-color:#ddd}
.badge{display:inline-flex;align-items:center;gap:6px;background:#2a2a2a;border:1px solid #444;border-radius:999px;padding:3px 10px;color:#ddd;font-size:12px;font-weight:500}
.result{border:1px solid #222;border-radius:16px;padding:16px;background:#161616}
.mono{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
.preview{white-space:pre-wrap;border:1px solid #222;background:#0f0f0f;border-radius:12px;padding:10px;max-height:200px;overflow:auto;color:#d6d6d6}
.scoreWrap{display:flex;align-items:center;gap:10px;margin-top:10px}
.scoreBar{width:180px;height:6px;background:#2a2a2a;border-radius:999px;overflow:hidden}
.scoreInner{height:6px;background:#eaeaea;border-radius:999px}
.err{border:1px solid #402828;background:#2a1515;color:#f3b4b4;border-radius:12px;padding:10px;font-size:14px}
.empty{color:#9a9a9a;font-size:14px;text-align:center;margin-top:20px}
.results{margin-top:20px;display:flex;flex-direction:column;gap:16px}
.auth-bar{display:flex;justify-content:space-between;align-items:center;padding:12px 20px;border-bottom:1px solid #222;background:#161616}
.auth-info{display:flex;align-items:center;gap:12px}
.avatar{width:32px;height:32px;border-radius:50%;border:1px solid #333}
.auth-btn{display:flex;align-items:center;gap:8px;padding:8px 16px;border:1px solid #333;border-radius:8px;background:#1c1c1c;color:#eee;text-decoration:none;cursor:pointer;font-size:14px}
.auth-btn:hover{background:#252525;color:#eee}
.login-card{max-width:400px;margin:100px auto;text-align:center}
.login-btn{display:inline-flex;align-items:center;gap:12px;padding:16px 24px;border:1px solid #333;border-radius:12px;background:#1c1c1c;color:#eee;text-decoration:none;font-size:16px;font-weight:600}
.login-btn:hover{background:#252525;color:#eee}
`;

export default function App() {
  // Start empty; no auto-search
  const [q, setQ] = useState("");
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<SimpleResult[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  // Authentication state
  const [authEnabled, setAuthEnabled] = useState<boolean | null>(null); // null = checking
  const [user, setUser] = useState<GitHubUser | null>(null);
  const [authLoading, setAuthLoading] = useState(true);
  const [token, setToken] = useState<string | null>(null);

  // Filters
  const [language, setLanguage] = useState("");
  const [pathContains, setPathContains] = useState("");
  const [repository, setRepository] = useState("");
  const [limit, setLimit] = useState(5);
  const [confidence, setConfidence] = useState(0.7); // New confidence filter

  // Available repositories
  const [repositories, setRepositories] = useState<string[]>([]);
  const [repositoriesLoading, setRepositoriesLoading] = useState(false);
  // Refs per-repo and selected ref
  const [refs, setRefs] = useState<string[]>([]);
  const [selectedRef, setSelectedRef] = useState<string>("");
  const [refsLoading, setRefsLoading] = useState(false);
  const [refsMap, setRefsMap] = useState<Record<string, string[]>>({});

  // File viewer state
  const [viewerFile, setViewerFile] = useState<{ path: string; content: string; repository: string; language?: string } | null>(null);

  const controllerRef = useRef<AbortController | null>(null);

  // Helper function to determine if repository is local
  const isLocalRepository = (repo: string) => {
    if (!repo) return false;
    // Local repos typically don't have protocol or domain
    return !repo.includes('://') && !repo.includes('github.com') && !repo.includes('.git');
  };

  // Function to open file preview in viewer
  const openFileViewer = (result: SimpleResult) => {
    setViewerFile({
      path: result.path,
      content: result.preview || 'No preview available',
      repository: result.repository || '',
      language: result.language
    });
  };

  // Function to close file viewer
  const closeFileViewer = () => {
    setViewerFile(null);
  };

  // Helper function to format repository name
  const formatRepoName = (repo: string) => {
    if (!repo) return '';

    // Remove protocol and .git suffix
    let formatted = repo.replace(/^https?:\/\//, '').replace(/\.git$/, '');

    // Extract org/name from github.com URLs
    if (formatted.includes('github.com/')) {
      const parts = formatted.split('/');
      const githubIndex = parts.findIndex(p => p === 'github.com');
      if (githubIndex >= 0 && parts.length > githubIndex + 2) {
        return `${parts[githubIndex + 1]}/${parts[githubIndex + 2]}`;
      }
    }

    // For other formats, try to extract last two path components
    const parts = formatted.split('/').filter(Boolean);
    if (parts.length >= 2) {
      return `${parts[parts.length - 2]}/${parts[parts.length - 1]}`;
    }

    return formatted;
  };

  // Function to update URL with current filter state
  const updateUrlWithFilters = () => {
    const url = new URL(window.location.href);
    const params = url.searchParams;

    // Clear existing filter params (but preserve other params)
    params.delete('r');
    params.delete('repository');
    params.delete('language');
    params.delete('lang');
    params.delete('path');
    params.delete('ref');

    // Set current filter values and query
    if (q.trim()) params.set('q', q.trim());
    if (repository) params.set('r', repository);
    if (language) params.set('language', language);
    if (pathContains) params.set('path', pathContains);
    if (selectedRef) params.set('ref', selectedRef);

    // Update URL without page reload
    const newUrl = `${url.pathname}${params.toString() ? '?' + params.toString() : ''}`;
    if (newUrl !== window.location.pathname + window.location.search) {
      window.history.replaceState({}, '', newUrl);
    }
  };

  // Fetch refs for a repository (server endpoint optional)
  async function fetchRefsForRepo(repoName: string) {
    if (!repoName) {
      setRefs([]);
      setSelectedRef("");
      return;
    }

    // If we already have refs in the map, use them
    if (refsMap[repoName]) {
      setRefs(refsMap[repoName]);
      return;
    }

    setRefsLoading(true);
    try {
      const headers: Record<string, string> = {};
      if (token) headers['Authorization'] = `Bearer ${token}`;
      // Try common REST pattern: /repositories/:repo/refs
      const r = await fetch(`${API_BASE}/repositories/${encodeURIComponent(repoName)}/refs`, {
        credentials: 'include',
        headers
      });
      if (r.ok) {
        const data = await r.json();
        const refList = Array.isArray(data) ? data.map((x: any) => String(x)) : [];
        setRefs(refList);
        setRefsMap(prev => ({ ...prev, [repoName]: refList }));
      } else {
        setRefs([]);
        setSelectedRef("");
      }
    } catch (e) {
      console.warn('Failed to fetch refs for', repoName, e);
      setRefs([]);
      setSelectedRef("");
    } finally {
      setRefsLoading(false);
    }
  }

  // Authentication functions
  const login = () => {
    window.location.href = `${API_BASE}/auth/github`;
  };

  const logout = async () => {
    try {
      await fetch(`${API_BASE}/auth/logout`, {
        method: 'POST',
        credentials: 'include'
      });
    } catch (e) {
      console.warn("Logout request failed:", e);
    }
    localStorage.removeItem('auth_token');
    setUser(null);
    setToken(null);
  };

  // Check authentication status on mount
  React.useEffect(() => {
    async function checkAuthStatus() {
      try {
        // First, check if auth is enabled
        const statusResponse = await fetch(`${API_BASE}/auth/status`);
        const statusData = await statusResponse.json();
        setAuthEnabled(statusData.enabled);

        if (!statusData.enabled) {
          // Auth is disabled, skip user auth check
          setAuthLoading(false);
          return;
        }

        // Auth is enabled, check if user is authenticated
        const storedToken = localStorage.getItem('auth_token');
        if (storedToken) {
          setToken(storedToken);
        }

        const response = await fetch(`${API_BASE}/auth/me`, {
          credentials: 'include',
          headers: storedToken ? { 'Authorization': `Bearer ${storedToken}` } : {}
        });

        if (response.ok) {
          const userData = await response.json();
          setUser(userData.user);
          if (userData.token && userData.token !== storedToken) {
            localStorage.setItem('auth_token', userData.token);
            setToken(userData.token);
          }
        } else {
          localStorage.removeItem('auth_token');
          setToken(null);
          setUser(null);
        }
      } catch (e) {
        console.warn("Auth check failed:", e);
        localStorage.removeItem('auth_token');
        setToken(null);
        setUser(null);
      } finally {
        setAuthLoading(false);
      }
    }
    checkAuthStatus();
  }, []);

  // Handle OAuth callback
  React.useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const code = urlParams.get('code');
    const state = urlParams.get('state');

    if (code && authEnabled) {
      async function handleCallback() {
        try {
          const response = await fetch(`${API_BASE}/auth/callback?code=${code}&state=${state || ''}`, {
            credentials: 'include'
          });

          if (response.ok) {
            const data = await response.json();
            setUser(data.user);
            if (data.token) {
              localStorage.setItem('auth_token', data.token);
              setToken(data.token);
            }
            window.history.replaceState({}, document.title, window.location.pathname);
          } else {
            setError('Authentication failed');
          }
        } catch (e) {
          setError('Authentication failed: ' + (e as Error).message);
        }
      }
      handleCallback();
    }
  }, [authEnabled]);

  // Initialize filters from URL parameters on component mount
  React.useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);

    // Set repository from URL if provided
    const repoParam = urlParams.get('r') || urlParams.get('repository');
    if (repoParam) {
      setRepository(repoParam);
    }
    // If no repo param in URL, explicitly ensure it stays empty
    else {
      setRepository("");
    }

    // Set ref from URL if provided - this needs to happen after refs are loaded
    const refParam = urlParams.get('ref');
    if (refParam) {
      setSelectedRef(refParam);
    }

    // You could also initialize other filters from URL if needed
    const langParam = urlParams.get('language') || urlParams.get('lang');
    if (langParam) {
      setLanguage(langParam);
    }
    const pathParam = urlParams.get('path');
    if (pathParam) {
      setPathContains(pathParam);
    }
    const queryParam = urlParams.get('q') || urlParams.get('query');
    if (queryParam) {
      setQ(queryParam);
    }
  }, []);

  // Fetch refs when repository changes (including on initial load)
  React.useEffect(() => {
    if (repository) {
      fetchRefsForRepo(repository);
    } else {
      setRefs([]);
      setSelectedRef(""); // Clear selected ref when repository is cleared
    }
  }, [repository, refsMap, token]); // Include refsMap and token as dependencies

  // Set ref from URL parameter after refs are loaded
  React.useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const refParam = urlParams.get('ref');

    if (refParam && refs.length > 0 && refs.includes(refParam)) {
      setSelectedRef(refParam);
    } else if (refParam && refs.length > 0 && !refs.includes(refParam)) {
      // Ref from URL is not available in current repository
      setSelectedRef("");
    }
  }, [refs]); // Run when refs are loaded/updated

  // Fetch available repositories on component mount
  React.useEffect(() => {
    async function fetchRepositories() {
      // If auth is enabled but user not authenticated, skip
      if (authEnabled && !user) return;

      setRepositoriesLoading(true);
      try {
        const headers: Record<string, string> = {};
        if (token) {
          headers['Authorization'] = `Bearer ${token}`;
        }

        const r = await fetch(`${API_BASE}/repositories`, {
          credentials: 'include',
          headers
        });
        if (r.ok) {
          const repos = await r.json();
          // repos can be an array of strings or objects { name, refs }
          if (Array.isArray(repos)) {
            const names: string[] = [];
            const map: Record<string, string[]> = {};
            for (const it of repos) {
              if (typeof it === 'string') {
                names.push(it);
              } else if (it && typeof it === 'object') {
                const name = it.name || it.repo || it.repository;
                if (name) {
                  names.push(name);
                  if (Array.isArray(it.refs)) map[name] = it.refs.map((r: any) => String(r));
                }
              }
            }
            setRepositories(names);
            if (Object.keys(map).length) setRefsMap(prev => ({ ...prev, ...map }));
          } else {
            setRepositories([]);
          }
        } else {
          // Clear list on non-ok (server may enforce auth)
          setRepositories([]);
        }
      } catch (e) {
        console.warn("Failed to fetch repositories:", e);
        setRepositories([]);
      } finally {
        setRepositoriesLoading(false);
      }
    }

    // Only fetch if auth status is determined
    if (authEnabled !== null) {
      fetchRepositories();
    }
  }, [authEnabled, user, token]);


  const queryString = useMemo(() => {
    const p = new URLSearchParams();
    p.set("q", q);
    p.set("k", String(limit));
    p.set("format", "simple");
    if (language) p.set("language", language);
    if (pathContains) p.set("path_contains", pathContains);
    if (selectedRef) p.set("ref", selectedRef);
    if (repository) p.set("repository", repository);
    return p.toString();
  }, [q, language, pathContains, repository, selectedRef, limit]);

  // Filter results client-side based on confidence threshold and limit to 20
  const filteredResults = useMemo(() => {
    return results.filter(r => r.score >= confidence).slice(0, 20);
  }, [results, confidence]);

  async function runSearch() {
    if (!q.trim()) return;

    // If auth is enabled and user is not authenticated, show error
    if (authEnabled && !user) {
      setError("Please log in to search");
      return;
    }

    // Update URL with current search parameters
    updateUrlWithFilters();

    setHasSearched(true);
    setLoading(true);
    setError(null);
    controllerRef.current?.abort();
    const ctrl = new AbortController();
    controllerRef.current = ctrl;
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      const r = await fetch(`${API_BASE}/search?${queryString}`, {
        signal: ctrl.signal,
        credentials: 'include',
        headers
      });
      const text = await r.text();
      if (!r.ok) throw new Error(text || `HTTP ${r.status}`);
      if (text.trim().startsWith("<")) throw new Error("API returned HTML – set VITE_API_BASE to your API host or add a dev proxy.");
      const data = JSON.parse(text) as SimpleResult[];
      setResults(Array.isArray(data) ? data : []);
    } catch (e: any) {
      if (e?.name !== "AbortError") setError(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  }

  // Show loading screen while checking authentication status
  if (authLoading) {
    return (
      <div className="app">
        <style>{styles}</style>
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
          <Loader2 width={32} height={32} style={{ animation: 'spin 1s linear infinite' }} />
        </div>
      </div>
    );
  }

  // Show login screen if auth is enabled and user is not authenticated
  if (authEnabled && !user) {
    return (
      <div className="app">
        <style>{styles}</style>
        <div className="login-card card">
          <h1 className="title" style={{ marginBottom: 16 }}>reposearch</h1>
          <p style={{ color: "#aaa", marginBottom: 32 }}>
            Natural language search of your codebase
          </p>
          <button className="login-btn" onClick={login}>
            <LogIn width={20} height={20} />
            Sign in with GitHub
          </button>
          {error && (
            <div className="err" style={{ marginTop: 16 }}>
              {error}
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="app">
      <style>{styles}</style>

      {/* Authentication bar - only show if auth is enabled */}
      {authEnabled && user && (
        <div className="auth-bar">
          <div className="auth-info">
            <img
              src={user.avatar_url}
              alt={user.name || user.login}
              className="avatar"
            />
            <span style={{ color: "#eee" }}>
              {user.name || user.login}
            </span>
          </div>
          <button className="auth-btn" onClick={logout}>
            <LogOut width={16} height={16} />
            Sign out
          </button>
        </div>
      )}

      <header className="header">
        <div className="container">
          <h1 className="title">reposearch</h1>
          <div className="subtitle">Natural language search of your codebase</div>
        </div>
      </header>

      <main className="container" style={{ paddingTop: 18 }}>
        {/* Hero search (centered, larger) */}
        <div className="card" style={{ margin: "0 auto 16px", maxWidth: 860 }}>
          <div className="row" style={{ justifyContent: "center" }}>
            <div className="input" style={{ maxWidth: "100%", width: 860 }}>
              <Search width={18} height={18} />
              <input
                placeholder="Search repo by intent (e.g., script that deletes disk)"
                value={q}
                onChange={(e) => setQ(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") runSearch();
                }}
              />
            </div>
            <button className="btn" onClick={runSearch} disabled={loading}>
              {loading ? <Loader2 width={18} height={18} /> : <Search width={18} height={18} />} Search
            </button>
          </div>

          <details className="filters" style={{ marginTop: 10 }}>
            <summary>
              <SlidersHorizontal width={16} height={16} /> Filters
            </summary>
            <div className="grid" style={{ gridTemplateColumns: "repeat(3, minmax(0, 1fr))" }}>
              <label className="row" style={{ alignItems: "center" }}>
                <span style={{ width: 110, color: "#999", fontSize: 12, textAlign: "right" }}>Repository</span>
                <select
                  className="select"
                  value={repository}
                  onChange={(e) => setRepository(e.target.value)}
                  disabled={repositoriesLoading}
                >
                  <option value="">Any</option>
                  {repositories.map((repo) => (
                    <option key={repo} value={repo}>
                      {repo}
                    </option>
                  ))}
                </select>
              </label>
              <label className="row" style={{ alignItems: "center" }}>
                <span style={{ width: 110, color: "#999", fontSize: 12, textAlign: "right" }}>Ref</span>
                <select className="select" value={selectedRef} onChange={(e) => setSelectedRef(e.target.value)} disabled={refsLoading || (!refs.length && !repository)}>
                  <option value="">Any</option>
                  {refs.map(r => (
                    <option key={r} value={r}>{r}</option>
                  ))}
                </select>
              </label>
              <label className="row" style={{ alignItems: "center" }}>
                <span style={{ width: 110, color: "#999", fontSize: 12, textAlign: "right" }}>Language</span>
                <select className="select" value={language} onChange={(e) => setLanguage(e.target.value)}>
                  <option value="">Any</option>
                  <option value="shell">shell</option>
                  <option value="python">python</option>
                  <option value="go">go</option>
                  <option value="markdown">markdown</option>
                  <option value="terraform">terraform</option>
                  <option value="yaml">yaml</option>
                  <option value="json">json</option>
                  <option value="javascript">javascript</option>
                  <option value="typescript">typescript</option>
                  <option value="ruby">ruby</option>
                  <option value="java">java</option>
                </select>
              </label>
              <label className="row" style={{ alignItems: "center" }}>
                <span style={{ width: 110, color: "#999", fontSize: 12, textAlign: "right" }}>Path contains</span>
                <input
                  className="text"
                  value={pathContains}
                  onChange={(e) => setPathContains(e.target.value)}
                  placeholder=""
                />
              </label>
              <label className="row" style={{ alignItems: "center" }}>
                <span style={{ width: 110, color: "#999", fontSize: 12, textAlign: "right" }}>Limit</span>
                <input
                  className="text"
                  type="number"
                  min={1}
                  max={20}
                  value={limit}
                  onChange={(e) => {
                    const value = parseInt(e.target.value || "10", 10);
                    setLimit(Math.min(Math.max(value, 1), 20));
                  }}
                  style={{ width: 90 }}
                />
              </label>
              <label className="row" style={{ alignItems: "center" }}>
                <span style={{ width: 110, color: "#999", fontSize: 12, textAlign: "right" }}>Min confidence</span>
                <input
                  className="text"
                  type="number"
                  min={0}
                  max={10}
                  step={0.1}
                  value={confidence}
                  onChange={(e) => setConfidence(parseFloat(e.target.value || "0.5"))}
                  style={{ width: 90 }}
                  placeholder="0.5"
                />
              </label>
            </div>
          </details>
        </div>

        {/* Results counter when filtered */}
        {hasSearched && !loading && results.length > 0 && (filteredResults.length !== results.length || filteredResults.length === 20) && (
          <div style={{ color: "#999", fontSize: 14, textAlign: "center", marginBottom: 12 }}>
            {(() => {
              const confidenceFiltered = results.filter(r => r.score >= confidence);
              const isLimited = confidenceFiltered.length > 20;
              const shownCount = filteredResults.length;
              const totalCount = results.length;

              if (isLimited) {
                return `Showing first ${shownCount} of ${confidenceFiltered.length} results (confidence ≥ ${confidence}) • Limited to 20`;
              } else {
                return `Showing ${shownCount} of ${totalCount} results (confidence ≥ ${confidence})`;
              }
            })()}
          </div>
        )}

        {/* Hints / Errors */}
        {!hasSearched && !loading && !error && (
          <div className="empty">Type a query above and press Enter or click Search.</div>
        )}
        {error && <div className="err" style={{ marginTop: 12 }}>{error}</div>}

        {/* Results */}
        <section style={{ display: "flex", flexDirection: "column", gap: 12, marginTop: 12 }}>
          {loading &&
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="result" style={{ opacity: 0.6 }}>
                <div style={{ height: 14, width: "50%", background: "#222", borderRadius: 6, marginBottom: 8 }} />
                <div style={{ height: 10, width: "35%", background: "#222", borderRadius: 6, marginBottom: 12 }} />
                <div style={{ height: 10, width: "100%", background: "#222", borderRadius: 6, marginBottom: 6 }} />
                <div style={{ height: 10, width: "80%", background: "#222", borderRadius: 6 }} />
              </div>
            ))}

          {hasSearched && !loading && !error && results.length === 0 && (
            <div className="empty">No results found. Try adjusting filters or wording.</div>
          )}

          {hasSearched && !loading && !error && results.length > 0 && filteredResults.length === 0 && (
            <div className="empty">
              No results meet the confidence threshold of {confidence}.
              <br />
              <button
                style={{
                  background: "transparent",
                  border: "1px solid #333",
                  color: "#999",
                  padding: "4px 8px",
                  borderRadius: "6px",
                  cursor: "pointer",
                  marginTop: "8px"
                }}
                onClick={() => setConfidence(0.1)}
              >
                Lower to 0.1
              </button>
            </div>
          )}

          {!loading &&
            filteredResults.map((r, idx) => {
              // Calculate relative score within the filtered result set
              const maxScore = Math.max(...filteredResults.map(res => res.score));
              const minScore = Math.min(...filteredResults.map(res => res.score));
              const scoreRange = maxScore - minScore;

              // Calculate percentage relative to the best result in this filtered set
              let pct: number;
              if (scoreRange === 0) {
                // All scores are the same
                pct = 1;
              } else {
                // Normalize to 0.2-1.0 range so even lower scores show some bar
                const normalizedScore = (r.score - minScore) / scoreRange;
                pct = 0.2 + (normalizedScore * 0.8);
              }

              const scorePct = `${Math.round(pct * 100)}%`;
              const scoreText = Number.isFinite(r.score) ? r.score.toFixed(3) : "0.000";

              return (
                <article key={idx} className="result">
                  <div style={{ display: "flex", justifyContent: "space-between", gap: 10 }}>
                    <div style={{ minWidth: 0 }}>
                      {isLocalRepository(r.repository || "") ? (
                        <button
                          onClick={() => openFileViewer(r)}
                          className="mono"
                          style={{
                            color: "#eaeaea",
                            textDecoration: "none",
                            wordBreak: "break-all",
                            background: "none",
                            border: "none",
                            cursor: "pointer",
                            textAlign: "left",
                            padding: 0,
                            font: "inherit"
                          }}
                        >
                          {r.path}
                        </button>
                      ) : (
                        <a
                          href={toGitHubUrl(r.path, r.line_start, r.line_end, r.repository || undefined, r.ref || undefined)}
                          target="_blank"
                          rel="noreferrer"
                          className="mono"
                          style={{ color: "#eaeaea", textDecoration: "none", wordBreak: "break-all" }}
                        >
                          {r.path}
                        </a>
                      )}
                    </div>

                    {isLocalRepository(r.repository || "") ? (
                      <button
                        onClick={() => openFileViewer(r)}
                        title="View preview"
                        style={{
                          color: "#cfcfcf",
                          textDecoration: "none",
                          whiteSpace: "nowrap",
                          background: "none",
                          border: "none",
                          cursor: "pointer",
                          padding: 0
                        }}
                      >
                        <ExternalLink width={16} height={16} />
                      </button>
                    ) : (
                      <a
                        href={toGitHubUrl(r.path, r.line_start, r.line_end, r.repository || undefined, r.ref || undefined)}
                        target="_blank"
                        rel="noreferrer"
                        title="Open on GitHub"
                        style={{ color: "#cfcfcf", textDecoration: "none", whiteSpace: "nowrap" }}
                      >
                        <ExternalLink width={16} height={16} />
                      </a>
                    )}
                  </div>

                  <div className="preview mono" style={{ marginTop: 8 }}>
                    {r.summary?.trim() ? r.summary : r.preview}
                  </div>

                  <div className="scoreWrap" style={{ justifyContent: "space-between" }}>
                    <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                      <div className="scoreBar">
                        <div className="scoreInner" style={{ width: scorePct }} />
                      </div>
                      <span style={{ fontSize: 12, color: "#aaa" }}>score {scoreText}</span>
                    </div>

                    {/* Repository and Ref badges - moved to right side of score */}
                    {(r.repository || r.ref) && (
                      <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                        {r.repository && (
                          <span className="badge" style={{ fontSize: 10 }}>
                            {formatRepoName(r.repository)}
                          </span>
                        )}
                        {r.ref && (
                          <span className="badge" style={{ fontSize: 10 }}>
                            {r.ref}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                </article>
              );
            })}
        </section>
      </main>

      {/* File Viewer Modal */}
      {viewerFile && (
        <div
          style={{
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: "rgba(0, 0, 0, 0.8)",
            zIndex: 1000,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            padding: 20
          }}
          onClick={closeFileViewer}
        >
          <div
            style={{
              backgroundColor: "#161616",
              border: "1px solid #333",
              borderRadius: 8,
              maxWidth: "90vw",
              maxHeight: "90vh",
              overflow: "hidden",
              display: "flex",
              flexDirection: "column"
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div
              style={{
                padding: "16px 20px",
                borderBottom: "1px solid #333",
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center"
              }}
            >
              <div>
                <div style={{ color: "#eee", fontWeight: "bold" }}>Preview: {viewerFile.path}</div>
                <div style={{ color: "#aaa", fontSize: "12px" }}>{viewerFile.repository} {viewerFile.language && `• ${viewerFile.language}`}</div>
              </div>
              <button
                onClick={closeFileViewer}
                style={{
                  background: "none",
                  border: "none",
                  color: "#aaa",
                  cursor: "pointer",
                  fontSize: "18px",
                  padding: 0,
                  width: 24,
                  height: 24,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center"
                }}
              >
                ×
              </button>
            </div>
            <div
              style={{
                padding: 20,
                overflow: "auto",
                flexGrow: 1,
                backgroundColor: "#0f0f0f"
              }}
            >
              <pre
                style={{
                  margin: 0,
                  color: "#eee",
                  fontSize: "13px",
                  fontFamily: "ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Monaco, Consolas, monospace",
                  lineHeight: 1.5,
                  whiteSpace: "pre-wrap",
                  wordWrap: "break-word"
                }}
              >
                {viewerFile.content}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
