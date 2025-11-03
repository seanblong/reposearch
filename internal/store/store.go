package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/seanblong/reposearch/pkg/models"
)

// Store provides methods to interact with the database.
type Store struct {
	pool *pgxpool.Pool
}

// ChunkStore defines the methods that the Store must implement.
type ChunkStore interface {
	GetRepositories(ctx context.Context) ([]string, error)
	Migrate(ctx context.Context, summaryDim int) error
	UpsertChunk(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error
	Search(ctx context.Context, summaryVec []float32, k int, opt QueryOpts) ([]models.SearchResult, error)
	GetChunkMeta(ctx context.Context, repository, path string, ls, le int) (ChunkMeta, bool, error)
}

// New creates a new Store instance connected to the given database URL.
func New(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Store{pool: p}, nil
}

func (s *Store) Close() { s.pool.Close() }

// GetRepositories returns a list of all unique repositories in the database.
func (s *Store) GetRepositories(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, "SELECT DISTINCT repository FROM chunks ORDER BY repository")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []string
	for rows.Next() {
		var repo string
		if err := rows.Scan(&repo); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	return repos, rows.Err()
}

// Migrate applies necessary database migrations and schema setup.
func (s *Store) Migrate(ctx context.Context, summaryDim int) error {
	q := `
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS chunks (
  id            TEXT PRIMARY KEY,
  repository    TEXT NOT NULL,
  ref           TEXT NOT NULL DEFAULT '',
  path          TEXT NOT NULL,
  language      TEXT,
  summary       TEXT,
  content       TEXT,
  line_start    INT,
  line_end      INT,
  summary_vec   vector(%d),
  content_hash  TEXT,
  summarized_at TIMESTAMP WITH TIME ZONE,
  created_at    TIMESTAMP WITH TIME ZONE DEFAULT now(),
  ts_fielded    tsvector GENERATED ALWAYS AS (
	setweight(
	  to_tsvector('english',
		regexp_replace(coalesce(path,''), '[^A-Za-z0-9]+', ' ', 'g')
	  ),
	  'A'
	) ||
	setweight(to_tsvector('english', coalesce(summary,'')), 'B') ||
	setweight(to_tsvector('english', coalesce(content,'')), 'C')
  ) STORED
);

CREATE UNIQUE INDEX IF NOT EXISTS chunks_repo_path_span_ref_uidx
  ON chunks (repository, ref, path, line_start, line_end);

CREATE INDEX IF NOT EXISTS chunks_repository_idx
  ON chunks (repository);

CREATE INDEX IF NOT EXISTS chunks_hash_idx
  ON chunks (content_hash);
CREATE INDEX IF NOT EXISTS chunks_ts_fielded_gin
  ON chunks USING GIN (ts_fielded);

CREATE INDEX IF NOT EXISTS chunks_summary_vec_idx
  ON chunks USING ivfflat (summary_vec vector_cosine_ops) WITH (lists = 100);
`
	_, err := s.pool.Exec(ctx, fmt.Sprintf(q, summaryDim))
	return err
}

// UpsertChunk inserts or updates a chunk.
func (s *Store) UpsertChunk(
	ctx context.Context,
	c models.Chunk,
	summaryVec []float32, // Only summary vector now
	contentHash string,
) error {
	var sv any
	if summaryVec != nil {
		sv = pgvector.NewVector(summaryVec)
	} else {
		sv = (*pgvector.Vector)(nil)
	}

	const q = `
		INSERT INTO chunks (
			id, repository, ref, path, language, summary, content,
			line_start, line_end, summary_vec, content_hash, summarized_at, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,
			CASE WHEN $6 <> '' THEN now() ELSE NULL END,
			now()
		)
		ON CONFLICT (repository, ref, path, line_start, line_end) DO UPDATE SET
			language     = EXCLUDED.language,
			content      = EXCLUDED.content,
			content_hash = EXCLUDED.content_hash,
			summary      = COALESCE(NULLIF(EXCLUDED.summary, ''), chunks.summary),
			summarized_at = COALESCE(EXCLUDED.summarized_at, chunks.summarized_at),
			summary_vec  = COALESCE(EXCLUDED.summary_vec, chunks.summary_vec),
			created_at   = chunks.created_at;`

	_, err := s.pool.Exec(ctx, q,
		c.ID, c.Repository, c.Ref, c.Path, c.Language, c.Summary, c.Content,
		c.LineStart, c.LineEnd, sv, contentHash,
	)
	return err
}

type QueryOpts struct {
	Repository   string // optional: filter by specific repository
	Ref          string // optional: filter by specific repository reference, e.g., branch
	Language     string // optional: "shell"|"python"|"go"|...
	PathContains string // optional substring filter
	QueryText    string // raw q for BM25/tsquery
}

func (s *Store) Search(
	ctx context.Context,
	summaryVec []float32, // Only one vector parameter now
	k int,
	opt QueryOpts,
) ([]models.SearchResult, error) {
	qtext := strings.TrimSpace(opt.QueryText)
	if qtext == "" {
		return []models.SearchResult{}, nil
	}

	sv := pgvector.NewVector(summaryVec)
	longest := longestToken(qtext)

	// Light "did they ask for scripts" nudge
	lq := strings.ToLower(qtext)
	askedForScript := strings.Contains(lq, "script") ||
		strings.Contains(lq, "scripts") ||
		strings.Contains(lq, "bash") ||
		strings.Contains(lq, "shell") ||
		strings.Contains(lq, "code") ||
		strings.Contains(lq, "program") ||
		strings.Contains(lq, "programs") ||
		strings.Contains(lq, "python") ||
		strings.Contains(lq, "cli")

	// Build params
	args := []any{
		sv,             // $1 summary vector
		qtext,          // $2 raw query text
		longest,        // $3 trigram token
		askedForScript, // $4 bool
	}
	ai := 5

	where := "TRUE"
	if opt.Repository != "" {
		where += fmt.Sprintf(" AND repository = $%d", ai)
		args = append(args, opt.Repository)
		ai++
	}
	if opt.Language != "" {
		where += fmt.Sprintf(" AND language = $%d", ai)
		args = append(args, opt.Language)
		ai++
	}
	if opt.PathContains != "" {
		where += fmt.Sprintf(" AND path ILIKE '%%' || $%d || '%%'", ai)
		args = append(args, opt.PathContains)
		ai++
	}
	if opt.Ref != "" {
		where += fmt.Sprintf(" AND ref = $%d", ai)
		args = append(args, opt.Ref)
		// Note: ai++ removed as it's not needed after this point
	}

	q := fmt.Sprintf(`
WITH parsed AS (
  SELECT lower(x) AS lx
  FROM ts_debug('english', $2) d, unnest(d.lexemes) AS x
  WHERE d.alias NOT IN ('StopWord','Space','Blank','Punct','Num')
),
terms AS (
  SELECT COALESCE(ARRAY_AGG(DISTINCT lx), ARRAY[]::text[]) AS all_terms
  FROM parsed
),
q AS (
  SELECT
    $1::vector AS sv,
    to_tsquery('english',
      (SELECT CASE WHEN cardinality(all_terms) > 0
                   THEN array_to_string(all_terms, ' | ')
                   ELSE NULL END
       FROM terms)
    ) AS tq_any,
    phraseto_tsquery('english',
      (SELECT CASE WHEN cardinality(all_terms) > 0
                   THEN array_to_string(all_terms, ' ')
                   ELSE NULL END
       FROM terms)
    ) AS tq_phrase,
    NULLIF($3,'') AS tri_term,
    $4::bool AS asked_script
),
cand AS (
  SELECT
    id, repository, ref, path, language, summary, content, line_start, line_end, created_at,

    -- Summary embedding similarity (now the primary signal)
    LEAST(GREATEST((1.0 - cosine_distance(summary_vec, (SELECT sv FROM q))), 0), 1) AS sem_sim,

    -- Lexical similarity of summary
    LEAST(GREATEST(
      ts_rank_cd(
        setweight(to_tsvector('english', coalesce(summary,'')), 'B'),
        (COALESCE((SELECT tq_any FROM q), ''::tsquery)
         || COALESCE((SELECT tq_phrase FROM q), ''::tsquery))
      ), 0), 1) AS lex_sum,
    -- Path trigram similarity
    COALESCE(similarity(lower(path), lower((SELECT tri_term FROM q))), 0) AS tri,

    -- Script bias
    CASE
      WHEN (SELECT asked_script FROM q) THEN
        CASE
          WHEN language IN ('shell','bash','sh','python','py','go') THEN 1
          WHEN language IN ('yaml','terraform','tf','json')         THEN -1
          ELSE 0
        END
      ELSE 0
    END AS script_bias,

    -- Noise penalty
    CASE
      WHEN lower(path) ~ '(?:(^|.*/))(sample|example|test|mock|fixture|tmp|temp|sandbox)(/|\\.|$)' THEN 1
      ELSE 0
    END AS noise_penalty
  FROM chunks
  WHERE %s
),
ranked AS (
  SELECT *,
         MAX(sem_sim) OVER()  AS max_sem,
         MAX(lex_sum) OVER()  AS max_lex,
         MAX(tri)     OVER()  AS max_tri
  FROM cand
)
SELECT
  id, repository, ref, path, language, summary, content, line_start, line_end, created_at,
  (
      0.80 * COALESCE(sem_sim / NULLIF(max_sem,0), 0) +
      0.15 * COALESCE(lex_sum / NULLIF(max_lex,0), 0) +
      0.05 * COALESCE(tri     / NULLIF(max_tri,0), 0) +
      0.10 * script_bias -
      0.07 * noise_penalty
  ) AS score
FROM ranked
ORDER BY score DESC
LIMIT %d;
`, where, k)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SearchResult
	for rows.Next() {
		var c models.Chunk
		var score float64
		if err := rows.Scan(
			&c.ID, &c.Repository, &c.Ref, &c.Path, &c.Language, &c.Summary, &c.Content, &c.LineStart, &c.LineEnd, &c.CreatedAt,
			&score,
		); err != nil {
			return nil, err
		}
		out = append(out, models.SearchResult{Chunk: c, Score: score})
	}
	return out, nil
}

// longestToken extracts the longest alphanumeric token from the input string.
func longestToken(s string) string {
	re := regexp.MustCompile(`[A-Za-z0-9._-]+`)
	toks := re.FindAllString(strings.ToLower(s), -1)
	longest := ""
	for _, t := range toks {
		if len(t) > len(longest) {
			longest = t
		}
	}
	return longest
}

// Ping checks the database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.pool.Ping(ctx)
}

// ChunkMeta holds metadata about a chunk.
type ChunkMeta struct {
	ContentHash   string
	Summary       string
	HasSummaryVec bool // Only summary vector now
}

// GetChunkMeta retrieves metadata for a chunk by repository, path and line span.
func (s *Store) GetChunkMeta(ctx context.Context, repository, path string, ls, le int) (ChunkMeta, bool, error) {
	const q = `
      SELECT content_hash,
             COALESCE(summary, ''),
             summary_vec IS NOT NULL
      FROM chunks
      WHERE repository = $1 AND path = $2 AND line_start = $3 AND line_end = $4
      LIMIT 1`
	var m ChunkMeta
	err := s.pool.QueryRow(ctx, q, repository, path, ls, le).
		Scan(&m.ContentHash, &m.Summary, &m.HasSummaryVec)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChunkMeta{}, false, nil
		}
		return ChunkMeta{}, false, err
	}
	return m, true, nil
}

// GetRefs returns distinct refs for a given repository.
func (s *Store) GetRefs(ctx context.Context, repository string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ref FROM chunks WHERE repository = $1 ORDER BY ref`, repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}
