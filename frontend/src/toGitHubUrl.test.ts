import { describe, it, expect } from 'vitest';
import { toGitHubUrl } from './github';

describe('toGitHubUrl', () => {
  it('builds a URL from owner/repo and path', () => {
    const url = toGitHubUrl('src/foo/bar.ts', undefined, undefined, 'owner/repo');
    expect(url).toContain('https://github.com/owner/repo/blob/');
    expect(url).toContain('src/foo/bar.ts');
  });

  it('handles full https path passed as repoPath', () => {
    const url = toGitHubUrl('https://raw.githubusercontent.com/owner/repo/main/README.md');
    expect(url).toContain('https://raw.githubusercontent.com/owner/repo/main/README.md');
  });

  it('converts git@ ssh clone URLs to https', () => {
    const url = toGitHubUrl('git@github.com:owner/repo/src/index.js');
    expect(url).toContain('https://github.com/owner/repo/src/index.js');
  });

  it('strips trailing .git', () =>
    expect(toGitHubUrl('src/a', undefined, undefined, 'owner/repo.git')).toContain('/owner/repo/'));
});
