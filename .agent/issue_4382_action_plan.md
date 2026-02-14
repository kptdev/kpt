# Action Plan: Issue #4382 - Move kpt docs from Porch to kpt

**🔗 Link**: https://github.com/kptdev/kpt/issues/4382  
**👤 Created by**: @liamfallon (Core Maintainer)  
**📅 Created**: Feb 10, 2026  
**🏷️ Labels**: `documentation`, `good first issue`, `cleanup`  
**⏱️ Estimated Time**: 4-8 hours

---

## 📋 **What This Issue Is About**

**Background**:
- Porch (Package Orchestration service) was originally part of kpt
- Porch was donated to the Nephio project (now at `nephio-project/porch`)
- During the split, kpt code was duplicated in Porch
- Porch team rewrote documentation for their cloned kpt code
- Now that code is being ported BACK to kpt (#4355 completed this)
- **We need to also move the NEW kpt-related documentation from Porch back to kpt**

**Your Task**:
Move the improved kpt-related documentation from `nephio-project/porch` repository to `kptdev/kpt` repository

---

## 🎯 **Step-by-Step Plan**

### **Phase 1: Research & Understand** (1-2 hours)

#### Step 1: Comment on the Issue to Claim It
```markdown
Hi @liamfallon! I'd like to work on this issue.

Before I start, could you clarify:
1. Which specific documentation files/sections from Porch should be moved to kpt?
2. Should I look in the Nephio porch repo (nephio-project/porch) or is there another location?
3. Are there any specific paths or folders I should focus on?

I'm ready to help move the kpt-related docs over!
```

**Why**: Get clarification from the maintainer before starting work

#### Step 2: Explore the Porch Repository
- **Repo**: https://github.com/nephio-project/porch
- **Look for**: Documentation related to kpt (not Porch-specific features)
- **Check directories**:
  - `/docs/`
  - `/documentation/`
  - `/README.md`
  - Any `/api/` or `/guides/` folders

#### Step 3: Explore the kpt Repository Documentation
- **Current docs location**: `c:\Users\Surbhi\Catroid\kpt\documentation\`
- **Understand structure**:
  - How docs are organized
  - Naming conventions
  - Link patterns

#### Step 4: Identify What to Move
Create a list of:
- Documentation files about kpt features (not Porch-specific)
- Updated API docs
- Improved guides/tutorials
- Any new examples

---

### **Phase 2: Move Documentation** (2-4 hours)

#### Step 5: Create a New Branch
```bash
cd c:\Users\Surbhi\Catroid\kpt
git checkout main
git pull origin main
git checkout -b docs/move-from-porch
```

#### Step 6: Copy Documentation Files
For each identified file:
1. Download from Porch repo
2. Place in appropriate location in kpt repo
3. Update frontmatter/metadata if needed
4. Update internal links to match kpt structure

#### Step 7: Update Links and References
- Fix any internal documentation links
- Update references to reflect kpt (not Porch)
- Ensure images/assets are included
- Check for broken links

#### Step 8: Test the Documentation Locally
```bash
# Navigate to documentation directory
cd documentation

# Install dependencies (if haven't already)
npm install

# Run local docs server
npm run start
```

**Verify**:
- All pages load correctly
- Links work
- Images/assets display
- No broken references

---

### **Phase 3: Clean Up & Submit** (1-2 hours)

#### Step 9: Write Clear Commit Messages
```bash
git add .
git commit -m "docs: move kpt-related documentation from Porch

Moved the following kpt-related documentation from nephio-project/porch:
- [List specific files/sections]

This documentation was rewritten in Porch and is now being moved
back to kpt following the code port in #4355.

Fixes #4382
```

#### Step 10: Create Pull Request
```bash
git push origin docs/move-from-porch
```

**PR Title**: `docs: Move kpt-related documentation from Porch`

**PR Description Template**:
```markdown
## Description

This PR moves kpt-related documentation from the Porch repository 
(nephio-project/porch) back to kpt, following the code port completed in #4355.

## Changes

### Documentation Moved:
- [ ] [File 1] - Brief description
- [ ] [File 2] - Brief description
- [ ] [etc.]

### Updates Made:
- [ ] Fixed internal links to match kpt structure
- [ ] Updated references from Porch to kpt
- [ ] Verified all images/assets are included
- [ ] Tested documentation builds locally

## Testing

- [x] Documentation builds without errors
- [x] All links work correctly
- [x] Images/assets display properly
- [x] Verified on local dev server

## Related Issues

Fixes #4382
Part of #4378

## Screenshots (if applicable)

[Add screenshots of doc pages if helpful]

## Checklist

- [ ] Documentation builds successfully
- [ ] All links verified
- [ ] No broken images
- [ ] Commit messages follow conventions
- [ ] PR description is clear
```

#### Step 11: Address Review Feedback
- Respond promptly to maintainer comments
- Make requested changes
- Re-test after updates

---

## 📚 **Resources You'll Need**

### Repositories:
- **Porch (Nephio)**: https://github.com/nephio-project/porch
- **kpt**: https://github.com/kptdev/kpt (you already have this)

### Related Issues:
- **#4378** (EPIC): https://github.com/kptdev/kpt/issues/4378
- **#4355** (Code port PR): https://github.com/kptdev/kpt/pull/4355

### kpt Documentation:
- **Live site**: https://kpt.dev
- **Local path**: `c:\Users\Surbhi\Catroid\kpt\documentation\`

---

## ⚠️ **Potential Challenges**

1. **Unclear which docs to move**
   - **Solution**: Ask maintainer for specific guidance in issue comment

2. **Documentation structure differences**
   - **Solution**: Study kpt docs structure first, adapt Porch docs to match

3. **Broken links after moving**
   - **Solution**: Use VSCode search to find/replace link patterns

4. **Build errors**
   - **Solution**: Test locally before submitting PR

---

## ✅ **Success Criteria**

- [ ] Identified all kpt-related docs from Porch
- [ ] Moved docs to appropriate locations in kpt
- [ ] All links work correctly
- [ ] Documentation builds without errors
- [ ] Images/assets included and working
- [ ] Clear commit messages
- [ ] PR approved and merged

---

## 🚀 **Ready to Start!**

### First Action: Post Comment on Issue

Copy this comment and post it on https://github.com/kptdev/kpt/issues/4382:

```
Hi @liamfallon! I'd like to work on this issue.

Before I start, could you clarify:
1. Which specific documentation files/sections from Porch should be moved to kpt?
2. Should I look in the Nephio porch repo (nephio-project/porch) or is there another location?
3. Are there any specific paths or folders I should focus on?

I'm ready to help move the kpt-related docs over! I'll start by exploring both repositories to identify candidates, but your guidance would help ensure I'm moving the right content.

Thanks!
```

---

**After posting the comment, while waiting for response**:
1. Explore https://github.com/nephio-project/porch/tree/main/docs
2. Compare with `c:\Users\Surbhi\Catroid\kpt\documentation\`
3. Create a draft list of candidate files to move

**Next Steps Document**: Once you get clarity, we'll create a detailed file-by-file migration plan!
