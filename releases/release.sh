COMMIT=$(git log -n1 --pretty='%h')
echo "Building reposearch with commit $COMMIT"

# Use VERSION if provided, otherwise try to get from git tag
if [ -n "$VERSION" ]; then
    TAG="$VERSION"
    echo "Using provided version: $TAG"
elif TAG=$(git describe --exact-match --tags $COMMIT); then
    echo "Building tag $TAG"
else
    echo "You need to checkout a tag to be able to build or provide VERSION"
    exit 1
fi

GOOSS=("darwin" "windows" "linux")
GOARCHS=("amd64" "arm64" "386")

mkdir -p downloads/$TAG

# Parse which component to build
COMPONENT_TO_BUILD=""
if [ -n "$COMPONENT" ]; then
    echo "Building component: $COMPONENT"
    COMPONENT_TO_BUILD="$COMPONENT"
else
    echo "No component specified, building all components"
    COMPONENT_TO_BUILD="all"
fi

# Build frontend if specified or building all
if [ "$COMPONENT_TO_BUILD" = "frontend" ] || [ "$COMPONENT_TO_BUILD" = "all" ]; then
    echo "\n\nBuilding frontend..."
    pushd frontend
    npm install
    npm run build
    popd
    tar -cvzf "downloads/$TAG/reposearch-frontend.$TAG.tar.gz" -C frontend dist package.json
else
    echo "Skipping frontend build"
fi

# Build Go binaries for specified component or all
for goos in "${GOOSS[@]}"; do
    for goarch in "${GOARCHS[@]}"; do
        # Mac/Darwin doesn't support 386 architecture
        if [ "$goos" == "darwin" ] && [ "$goarch" == "386" ]; then
            continue
        fi

        # Build API if specified or building all
        if [ "$COMPONENT_TO_BUILD" = "api" ] || [ "$COMPONENT_TO_BUILD" = "all" ]; then
            echo "\n\nbuilding api $goos $goarch"
            mkdir reposearch-api
            pushd reposearch-api
            GOOS=$goos GOARCH=$goarch go build -ldflags "-X main.version=$TAG" github.com/seanblong/reposearch/cmd/api
            popd
            tar -cvzf "downloads/$TAG/reposearch-api.$TAG.$goos.$goarch.tar.gz" reposearch-api
            rm -rf reposearch-api
        fi

        # Build Indexer if specified or building all
        if [ "$COMPONENT_TO_BUILD" = "indexer" ] || [ "$COMPONENT_TO_BUILD" = "all" ]; then
            echo "\n\nbuilding indexer $goos $goarch"
            mkdir reposearch-indexer
            pushd reposearch-indexer
            GOOS=$goos GOARCH=$goarch go build -ldflags "-X main.version=$TAG" github.com/seanblong/reposearch/cmd/indexer
            popd
            tar -cvzf "downloads/$TAG/reposearch-indexer.$TAG.$goos.$goarch.tar.gz" reposearch-indexer
            rm -rf reposearch-indexer
        fi
    done
done
