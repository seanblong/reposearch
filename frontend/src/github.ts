export function toGitHubUrl(repoPath: string, lineStart?: number, lineEnd?: number, repoOverride?: string, refOverride?: string) {
  let rp = repoPath.trim();
  const httpIdx = rp.search(/https?:\/\//i);
  if (httpIdx >= 0) {
    rp = rp.slice(httpIdx);
  } else if (/^git@github\.com:/.test(rp)) {
    rp = rp.replace(/^git@github\.com:/, 'https://github.com/');
  }
  rp = rp.replace(/\.git(?=\/|$)/, '');

  if (/^https?:\/\//.test(rp)) {
    const hash = lineStart ? `#L${lineStart}` + (lineEnd ? `-L${lineEnd}` : '') : '';
    return `${rp}${hash}`;
  }

  const env = (import.meta as any)?.env || {};
  let githubBaseFromEnv = env.VITE_GITHUB_BASE;
  let repo = env.GIT_REPO || 'seanblong/reposearch';
  if (repoOverride) {
    try {
      const cleaned = repoOverride.trim();
      if (/^https?:\/\//.test(cleaned) || cleaned.includes('github.com')) {
        const withoutProto = cleaned.replace(/^https?:\/\//, '');
        const parts = withoutProto.split('/').filter(Boolean);
        const idx = parts.indexOf('github.com');
        const ownerIdx = idx >= 0 ? idx + 1 : 0;
        if (parts.length > ownerIdx) {
          const owner = parts[ownerIdx];
          const repoName = (parts[ownerIdx + 1] || '').replace(/\.git$/, '');
          if (owner && repoName) repo = `${owner}/${repoName}`;
        }
      } else {
        const parts = repoOverride.split('/').filter(Boolean);
        if (parts.length >= 2) {
          const owner = parts[0];
          const repoName = parts[1].replace(/\.git$/, '');
          repo = `${owner}/${repoName}`;
        } else {
          repo = repoOverride.replace(/\.git$/, '');
        }
      }
    } catch (e) {
      repo = repoOverride.replace(/\.git$/, '');
    }
  }

  // Use refOverride if provided, otherwise fall back to environment or 'main'
  const branch = refOverride || env.GIT_REF || 'main';

  if (githubBaseFromEnv && !/^https?:\/\//.test(githubBaseFromEnv)) {
    githubBaseFromEnv = `https://${githubBaseFromEnv}`;
  }

  const base = githubBaseFromEnv || `https://github.com/${repo}/blob/${branch}`;
  const hash = lineStart ? `#L${lineStart}` + (lineEnd ? `-L${lineEnd}` : '') : '';
  const joined = `${base.replace(/\/+$/, '')}/${rp.replace(/^\/+/, '')}`;
  return `${joined}${hash}`;
}

export default toGitHubUrl;
