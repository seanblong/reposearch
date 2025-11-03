# reposearch
[![Coverage](https://img.shields.io/badge/Coverage-52.2%25-yellow)](https://github.com/seanblong/reposearch/actions/workflows/test.yaml)

Repo Search provides natural language search for your repositories.  Rather than
relying on keyword matching, it uses vector embeddings to find relevant code and
documentation based on semantic meaning.

## 🚀 Quickstart

Set your AI provider API key and project.

```console
export REPOSEARCH_PROVIDER_API_KEY="your-api-key-here"
export REPOSEARCH_PROVIDER_PROJECT_ID="your-project-id"
export REPOSEARCH_PROVIDER="openai"
```

At this time, `reposearch` supports **openai** and **vertexai** as embedding and
generation providers. See [config/reposearch.yaml](config/reposearch.yaml) for
full configuration options.

Run with Docker Compose:

```console
docker-compose up
```

Navigate to `http://localhost:3000` to access the web UI.

To stop the application, run:

```console
docker-compose down
```

## 💻 Developing

Minimally, set the following environment variables to get started with OpenAI:

```bash
export REPOSEARCH_DB_URL="postgres://user:password@localhost:5432/reposearch?sslmode=disable"
export REPOSEARCH_PROVIDER="openai"
export REPOSEARCH_PROVIDER_API_KEY="your_openai_api_key_here"
export REPOSEARCH_PROVIDER_PROJECT_ID="your_openai_project_id_here"
export REPOSEARCH_GIT_REPO="https://github.com/your-org/your-repo"
```

See [config/reposearch.yaml](config/reposearch.yaml) for full configuration options.

Spin up a local Postgres instance with pgvector:

```bash
docker run -d --name reposearch-db -p 5432:5432 -e POSTGRES_PASSWORD=postgres \
    -e POSTGRES_DB=reposearch ankane/pgvector
```

Run the indexer:

```bash
go run cmd/indexer/main.go
```

> [!NOTE]
> On initial indexing this can take some time, but subsequent indexing will only
> index deltas.

Run the API server:

```bash
go run cmd/api/main.go
```

Run the web frontend:

```bash
cd frontend
npm install
npm run dev
```

Navigate to `http://localhost:5173` to access the web UI.

## 🛠️ Building

To build the `reposearch` binaries:

```bash
go build -o indexer ./cmd/indexer
go build -o reposearch-api ./cmd/api
```

To build the Docker images:

```bash
docker build -f Dockerfile.api -t reposearch-api .
docker build -f Dockerfile.frontend -t reposearch-frontend .
docker build -f Dockerfile.indexer -t reposearch-indexer .
```

## 🔐 Authentication

TBD

## 🙏 Acknowledgments

The code in this project was largely authored by generative AI models:

- OpenAI — GPT-5
- Google — Gemini 2.5
- Anthropic — Claude Sonnet 4

All AI-generated material was reviewed and adapted by the project maintainers and
is licensed under this repository's license.

## 🤝 Contributing

Contributions are welcome! Please see the [CONTRIBUTING.md](CONTRIBUTING.md) file
for more information.
