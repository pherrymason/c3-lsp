if [ -z "$VERSION" ]; then
  echo "VERSION is not set. Usage: make index-c3-std VERSION=x.y.z"
  exit 1
fi

echo "C3C_DIR: $C3C_DIR"
echo "VERSION: $VERSION"
cd "$C3C_DIR" && git fetch --all && git reset --hard origin/master && git checkout tags/v"$VERSION"
cd ../../server/cmd/stdlib_indexer && go run main.go blurp.go --"$VERSION"