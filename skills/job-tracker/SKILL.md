# Job Tracker Skill

Track and manage job applications, interviews, and career progress in your Obsidian vault.

## When to Use

Use this skill when the user wants to:
- Track job applications
- Log interview stages and notes
- Manage job search progress
- Get overview of their job search status

## Vault Structure

Your job tracking system:

```
20-Career/
├── 21-Applications/
│   ├── 2026-Spring/          # Current applications
│   │   ├── Ada-DevOps.md
│   │   ├── Apple-Expert.md
│   │   └── ...
│   └── Archive/              # Past applications
├── 22-Resumes/               # Master + tailored resumes
│   ├── master_resume.tex
│   ├── master_resume.pdf
│   ├── Master-Index.md
│   ├── Generated/
│   ├── ada_DevOps/
│   ├── Apple_Expert/
│   └── ...
├── 23-LinkedIn/             # LinkedIn strategies
├── 24-Interview-Prep/       # Interview prep notes
├── 25-Company-Research/     # Company research
└── 26-Salary-Tracker/      # Salary tracking
```

## Job Application Note Format

Each job application note should link to its tailored resume:

```markdown
# Job Application: Company - Role

**Status:** Applied
**Date Applied:** Feb 2026
**Source:** Greenhouse

## Resume Used
[[../22-Resumes/company-role/resume.pdf]]

## Timeline
- [x] Application Submitted
- [ ] Initial Screen
- [ ] Technical Interview
- [ ] Final Interview
- [ ] Offer Received

## Notes
-

## Links
- [Job Posting](url)
```

## How to Use

1. **Add new job**: Create note in `21-Applications/2026-Spring/`
2. **Create tailored resume**: Use resume-tailor skill, save to `22-Resumes/{company}-{role}/`
3. **Link resume**: Add link in job note: `[[../22-Resumes/company-role/resume.pdf]]`
4. **Update status**: Edit the Status field
5. **Log interview**: Add notes to the Notes section

## Examples

- "I applied to a new job at Google" → create note + tailored resume in 22-Resumes/Google-SWE/
- "What's my job search progress?" → list all notes in 21-Applications
- "Show me all jobs I've applied to" → search_content for "Status: Applied"
