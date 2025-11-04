COMMIT=$(git log -n1 --pretty='%h')
echo "Building reposearch with commit $COMMIT"
if TAG=$(git describe --exact-match --tags $COMMIT); then
    echo "Building tag $TAG"
else
    echo "You need to checkout a tag to be able to build"
    exit 1
fi

GOOSS=("darwin" "windows" "linux")
GOARCHS=("amd64" "arm64" "386")

mkdir -p downloads/$TAG

# Build frontend
echo "\n\nBuilding frontend..."
pushd frontend
npm install
npm run build
popd
tar -cvzf "downloads/$TAG/reposearch-frontend.$TAG.tar.gz" -C frontend dist package.json

for goos in "${GOOSS[@]}"; do
    for goarch in "${GOARCHS[@]}"; do
        # Mac/Darwin doesn't support 386 architecture
        if [ "$goos" == "darwin" ] && [ "$goarch" == "386" ]; then
            continue
        fi
        echo "\n\nbuilding $goos $goarch"
        mkdir reposearch-api
        pushd reposearch-api
        GOOS=$goos GOARCH=$goarch go build -ldflags "-X main.version=$TAG" github.com/seanblong/reposearch/cmd/api
        popd
        tar -cvzf "downloads/$TAG/reposearch-api.$TAG.$goos.$goarch.tar.gz" reposearch-api
        rm -rf reposearch-api

        echo "\n\nbuilding $goos $goarch"
        mkdir reposearch-indexer
        pushd reposearch-indexer
        GOOS=$goos GOARCH=$goarch go build -ldflags "-X main.version=$TAG" github.com/seanblong/reposearch/cmd/indexer
        popd
        tar -cvzf "downloads/$TAG/reposearch-indexer.$TAG.$goos.$goarch.tar.gz" reposearch-indexer
        rm -rf reposearch-indexer
    done
done
