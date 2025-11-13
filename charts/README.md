# reposearch

The [reposearch](./reposearch) chart deploys the reposearch application, composed
of a frontend, backend (API), database, and an indexer (with some list of jobs
based on repos to index).

The frontend and backend subcharts are based on the [generic](./generic) chart
available in this directory.  That chart was lifted from [Bitnami's Helm
template](https://github.com/bitnami/charts/tree/main/template/CHART_NAME) and
should be fairly extendable with regards to overrides.  Please note, however, that
the generic hasn't been tested much outside the current implementation in this chart,
and may need alterations.
