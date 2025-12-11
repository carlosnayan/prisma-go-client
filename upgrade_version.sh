#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to validate version format (semver)
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9-]+)?(\.[0-9]+)?$ ]]; then
        echo -e "${RED}Error: Invalid version. Use semver format (e.g., 0.1.3)${NC}" >&2
        exit 1
    fi
}

# Get new version
if [ -z "$1" ]; then
    # If no version provided, read from VERSION file
    if [ ! -f "VERSION" ]; then
        echo -e "${RED}Error: VERSION file not found and no version was provided${NC}" >&2
        echo "Usage: $0 [new-version]" >&2
        exit 1
    fi
    NEW_VERSION=$(cat VERSION | tr -d '[:space:]')
    echo -e "${YELLOW}Reading version from VERSION file: ${NEW_VERSION}${NC}"
else
    NEW_VERSION=$1
fi

# Validate version
validate_version "$NEW_VERSION"

# Get current version
CURRENT_VERSION=$(grep -h "0\.1\.[0-9]" cmd/prisma/cmd/root.go cmd/prisma/cmd/generate.go prisma.go 2>/dev/null | head -1 | grep -oE "0\.1\.[0-9]+" | head -1)

if [ -z "$CURRENT_VERSION" ]; then
    CURRENT_VERSION=$(grep -E "^[0-9]+\.[0-9]+\.[0-9]+" VERSION 2>/dev/null | head -1 | tr -d '[:space:]' || echo "unknown")
fi

echo -e "${GREEN}Updating version from ${CURRENT_VERSION} to ${NEW_VERSION}${NC}"
echo ""

# Files to update
FILES=(
    "VERSION"
    "cmd/prisma/cmd/root.go"
    "cmd/prisma/cmd/generate.go"
    "prisma.go"
)

# Update each file
for file in "${FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo -e "${YELLOW}Warning: File $file not found, skipping...${NC}"
        continue
    fi

    case "$file" in
        "VERSION")
            echo "$NEW_VERSION" > "$file"
            echo -e "${GREEN}✓${NC} Updated $file"
            ;;
        "cmd/prisma/cmd/root.go")
            # Update line: "0.1.0", -> "0.1.3",
            sed -i '' -E "s/\"[0-9]+\.[0-9]+\.[0-9]+\"/\"$NEW_VERSION\"/g" "$file"
            echo -e "${GREEN}✓${NC} Updated $file"
            ;;
        "cmd/prisma/cmd/generate.go")
            # Update: const version = "0.1.0"
            sed -i '' -E "s/const version = \"[0-9]+\.[0-9]+\.[0-9]+\"/const version = \"$NEW_VERSION\"/g" "$file"
            echo -e "${GREEN}✓${NC} Updated $file"
            ;;
        "prisma.go")
            # Update: const Version = "0.1.0"
            sed -i '' -E "s/const Version = \"[0-9]+\.[0-9]+\.[0-9]+\"/const Version = \"$NEW_VERSION\"/g" "$file"
            echo -e "${GREEN}✓${NC} Updated $file"
            ;;
    esac
done

echo ""
echo -e "${GREEN}Version successfully updated to ${NEW_VERSION}!${NC}"
echo ""
echo "Updated files:"
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "  - $file"
    fi
done
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Commit changes: git commit -am \"chore: bump version to $NEW_VERSION\""
echo "  3. Create tag: git tag -a v$NEW_VERSION -m \"Release v$NEW_VERSION\""
echo "  4. Push tag: git push origin v$NEW_VERSION"