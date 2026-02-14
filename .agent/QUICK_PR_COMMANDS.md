# Quick PR Creation Guide

## Run these commands in order:

# 1. Create branch
git checkout -b feat/cel-conditional-execution

# 2. Add all changes
git add .

# 3. Commit with descriptive message
git commit -m "feat: Add CEL-based conditional function execution

Implements #4388

- Add 'condition' field to Function schema for CEL expressions
- Integrate google/cel-go library for condition evaluation  
- Skip function execution when condition evaluates to false
- Add comprehensive unit and E2E tests

Functions can now be conditionally executed based on package contents:

pipeline:
  mutators:
  - image: set-namespace:v0.4
    condition: \"resources.exists(r, r.kind == 'ConfigMap')\"
"

# 4. Push to your fork
git push origin feat/cel-conditional-execution

# 5. Go to GitHub and create PR!
# https://github.com/kptdev/kpt/compare/main...YOUR_USERNAME:feat/cel-conditional-execution
