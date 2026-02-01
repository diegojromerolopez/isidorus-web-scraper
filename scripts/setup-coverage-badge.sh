#!/bin/bash

# Coverage Badge Setup Script
# This script helps you set up the coverage badge for Isidorus Web Scraper

set -e

echo "=================================================="
echo "  Coverage Badge Setup for Isidorus Web Scraper"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Step 1: Create GitHub Gist
echo -e "${BLUE}Step 1: Create GitHub Gist${NC}"
echo "-----------------------------------"
echo "We need to create a GitHub Gist to store the coverage badge data."
echo ""
echo "Please follow these steps:"
echo "  1. Open: https://gist.github.com/"
echo "  2. Click 'New gist' (or you may see the creation form directly)"
echo "  3. Fill in the following:"
echo "     - Filename: isidorus-coverage.json"
echo "     - Content (copy this):"
echo ""
echo '{"schemaVersion": 1, "label": "coverage", "message": "0%", "color": "red"}'
echo ""
echo "  4. Make sure it's set to 'Create public gist'"
echo "  5. Click 'Create public gist'"
echo "  6. Copy the Gist ID from the URL (e.g., https://gist.github.com/username/GIST_ID)"
echo ""
echo -e "${YELLOW}Opening GitHub Gist in your browser...${NC}"
open "https://gist.github.com/" 2>/dev/null || echo "Please manually open: https://gist.github.com/"
echo ""
read -p "Press Enter once you've created the gist and copied the Gist ID..."
echo ""
read -p "Enter your Gist ID: " GIST_ID

if [ -z "$GIST_ID" ]; then
    echo -e "${RED}Error: Gist ID cannot be empty${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Gist ID saved: $GIST_ID${NC}"
echo ""

# Step 2: Create Personal Access Token
echo -e "${BLUE}Step 2: Create Personal Access Token${NC}"
echo "-----------------------------------"
echo "We need a GitHub Personal Access Token with 'gist' scope."
echo ""
echo "Please follow these steps:"
echo "  1. Open: https://github.com/settings/tokens"
echo "  2. Click 'Generate new token' → 'Generate new token (classic)'"
echo "  3. Give it a name: 'Coverage Badge Token'"
echo "  4. Select ONLY the 'gist' scope (check the box)"
echo "  5. Click 'Generate token' at the bottom"
echo "  6. Copy the token (you won't be able to see it again!)"
echo ""
echo -e "${YELLOW}Opening GitHub Token settings in your browser...${NC}"
open "https://github.com/settings/tokens" 2>/dev/null || echo "Please manually open: https://github.com/settings/tokens"
echo ""
read -p "Press Enter once you've created the token and copied it..."
echo ""
read -sp "Enter your Personal Access Token: " GIST_SECRET
echo ""

if [ -z "$GIST_SECRET" ]; then
    echo -e "${RED}Error: Token cannot be empty${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Token saved${NC}"
echo ""

# Step 3: Add GitHub Secrets
echo -e "${BLUE}Step 3: Add GitHub Repository Secrets${NC}"
echo "-----------------------------------"
echo "We need to add two secrets to your GitHub repository."
echo ""
echo "Please follow these steps:"
echo "  1. Open: https://github.com/diegojromerolopez/isidorus-web-scraper/settings/secrets/actions"
echo "  2. Click 'New repository secret'"
echo "  3. Add the first secret:"
echo "     - Name: GIST_SECRET"
echo "     - Value: [paste the token you just created]"
echo "  4. Click 'Add secret'"
echo "  5. Click 'New repository secret' again"
echo "  6. Add the second secret:"
echo "     - Name: GIST_ID"
echo "     - Value: $GIST_ID"
echo "  7. Click 'Add secret'"
echo ""
echo -e "${YELLOW}Opening GitHub Secrets settings in your browser...${NC}"
open "https://github.com/diegojromerolopez/isidorus-web-scraper/settings/secrets/actions" 2>/dev/null || echo "Please manually open: https://github.com/diegojromerolopez/isidorus-web-scraper/settings/secrets/actions"
echo ""
echo "Here are your values to paste:"
echo -e "${YELLOW}GIST_ID:${NC} $GIST_ID"
echo -e "${YELLOW}GIST_SECRET:${NC} [the token you just created]"
echo ""
read -p "Press Enter once you've added both secrets to GitHub..."
echo ""

# Step 4: Update README.md
echo -e "${BLUE}Step 4: Update README.md${NC}"
echo "-----------------------------------"
echo "Updating README.md with your Gist ID..."
echo ""

README_FILE="../README.md"
if [ -f "$README_FILE" ]; then
    # Create backup
    cp "$README_FILE" "${README_FILE}.backup"
    echo -e "${GREEN}✓ Created backup: ${README_FILE}.backup${NC}"
    
    # Replace GIST_ID in README
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/GIST_ID/$GIST_ID/g" "$README_FILE"
    else
        # Linux
        sed -i "s/GIST_ID/$GIST_ID/g" "$README_FILE"
    fi
    
    echo -e "${GREEN}✓ Updated README.md with Gist ID${NC}"
else
    echo -e "${RED}Error: README.md not found${NC}"
    exit 1
fi
echo ""

# Step 5: Verify and Commit
echo -e "${BLUE}Step 5: Verify and Commit${NC}"
echo "-----------------------------------"
echo "Let's verify the changes and commit them."
echo ""

# Show the diff
echo "Changes to README.md:"
git diff "$README_FILE" || true
echo ""

read -p "Do you want to commit these changes? (y/n): " COMMIT_CHOICE

if [ "$COMMIT_CHOICE" = "y" ] || [ "$COMMIT_CHOICE" = "Y" ]; then
    git add "$README_FILE"
    git commit -m "Add coverage badge with Gist ID"
    echo -e "${GREEN}✓ Changes committed${NC}"
    echo ""
    
    read -p "Do you want to push to GitHub? (y/n): " PUSH_CHOICE
    
    if [ "$PUSH_CHOICE" = "y" ] || [ "$PUSH_CHOICE" = "Y" ]; then
        git push
        echo -e "${GREEN}✓ Changes pushed to GitHub${NC}"
        echo ""
        echo -e "${GREEN}=================================================="
        echo "  🎉 Coverage Badge Setup Complete! 🎉"
        echo "==================================================${NC}"
        echo ""
        echo "The coverage badge will be updated automatically on the next push to main."
        echo "You can view your repository at:"
        echo "https://github.com/diegojromerolopez/isidorus-web-scraper"
    else
        echo -e "${YELLOW}Changes committed but not pushed. Run 'git push' when ready.${NC}"
    fi
else
    echo -e "${YELLOW}Changes not committed. You can review them with 'git diff'${NC}"
    echo "To restore the original README: mv ${README_FILE}.backup ${README_FILE}"
fi

echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "  1. The workflow will run automatically on the next push to main"
echo "  2. Check the Actions tab to see the coverage badge being generated"
echo "  3. The badge should appear in your README within a few minutes"
echo ""
echo "For troubleshooting, see: docs/COVERAGE_BADGE_SETUP.md"
