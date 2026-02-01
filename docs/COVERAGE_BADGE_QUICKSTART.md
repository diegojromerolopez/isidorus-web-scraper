# Test Coverage Badge - Quick Start

## What You Need to Do

To enable the coverage badge in your README, follow these 3 simple steps:

### 1. Create a GitHub Gist

Visit https://gist.github.com/ and create a new gist:
- **Filename**: `isidorus-coverage.json`
- **Content**:
```json
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "0%",
  "color": "red"
}
```
- Make it **public**
- Copy the **Gist ID** from the URL (the long alphanumeric string)

### 2. Create a GitHub Personal Access Token

1. Go to: https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Select scope: **gist**
4. Generate and copy the token

### 3. Add Repository Secrets

In your repository settings (Settings → Secrets and variables → Actions):
- Add secret `GIST_SECRET` = your token from step 2
- Add secret `GIST_ID` = your Gist ID from step 1

### 4. Update README.md

Replace `GIST_ID` in line 8 of README.md with your actual Gist ID:

```markdown
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/diegojromerolopez/YOUR_ACTUAL_GIST_ID/raw/isidorus-coverage.json)](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/tests-unit.yml)
```

### 5. Push and Wait

Push to `main` branch and the badge will update automatically! 🎉

---

For detailed instructions, see [COVERAGE_BADGE_SETUP.md](./COVERAGE_BADGE_SETUP.md)
