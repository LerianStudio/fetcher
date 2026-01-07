#!/bin/bash

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Output directory (default: ./artifacts)
OUTPUT_DIR="${1:-./artifacts}"
mkdir -p "$OUTPUT_DIR"

echo "${BLUE}Generating test coverage report...${NC}"

PACKAGES=$(go list ./... | grep -v -f ./scripts/coverage_ignore.txt)

echo "${BLUE}Running tests on packages:${NC}"
echo "$PACKAGES"

go test -cover $PACKAGES -coverprofile="$OUTPUT_DIR/coverage.raw.out"

# Filter out mock files from coverage (they are generated and should not be counted)
echo "${BLUE}Filtering out .mock.go files from coverage...${NC}"
grep -v '\.mock\.go' "$OUTPUT_DIR/coverage.raw.out" > "$OUTPUT_DIR/coverage.out" || cp "$OUTPUT_DIR/coverage.raw.out" "$OUTPUT_DIR/coverage.out"

printf "\n${GREEN}Coverage Summary (excluding .mock.go files):${NC}\n"
go tool cover -func="$OUTPUT_DIR/coverage.out"

printf "\n${BLUE}Generating HTML coverage report...${NC}\n"
go tool cover -html="$OUTPUT_DIR/coverage.out" -o "$OUTPUT_DIR/coverage.html"
echo "${GREEN}HTML coverage report generated at: ${NC}$OUTPUT_DIR/coverage.html"

# Cleanup
rm -f "$OUTPUT_DIR/coverage.raw.out"
