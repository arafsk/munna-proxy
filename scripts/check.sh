#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

echo "==> gofmt"
unformatted=""
while IFS= read -r file; do
  result="$(gofmt -l "$file")"
  if [[ -n "$result" ]]; then
    unformatted+="$result"$'\n'
  fi
done < <(git ls-files --cached --others --exclude-standard '*.go')
if [[ -n "$unformatted" ]]; then
  echo "unformatted files:"
  echo "$unformatted"
  exit 1
fi

echo "==> go vet"
go vet ./...

echo "==> golangci-lint"
golangci-lint run

echo "==> go test"
go test -race -count=1 ./...
