# Resume Tailor Skill

Tailor your resume for specific job applications using your existing resume and job requirements.

## When to Use

Use this skill when the user wants to:
- Tailor their resume for a specific job
- Highlight relevant experience for an application
- Update their master resume
- Create a targeted resume from their experience

## Vault Structure

Your resume system is in the Obsidian vault:
- `20-Career/22-Resumes/` - Master resume and generated versions
- `20-Career/21-Applications/` - Job applications
- `20-Career/24-Interview-Prep/` - Interview preparation

The vault is synced via obsidian-headless on your homelab.

## Master Resume Location

The master resume files are stored at:
- `~/Documents/resumes/resume.tex` - LaTeX source
- `~/Documents/resumes/resume.pdf` - Compiled PDF

These sync to the vault via Obsidian Sync.

## Tools

Use the obsidian-vault MCP to:
- `read_file` - Read existing resume or job description
- `write_file` - Create/update tailored resumes
- `search_content` - Find relevant experience in other notes
- `list_directory` - List resume files

## How to Tailor a Resume

1. **Get job description**: Read the job posting or create a note with requirements
2. **Read master resume**: Load the user's main resume
3. **Match keywords**: Find matching skills and experiences
4. **Create tailored version**: Write a new resume highlighting relevant points

## Resume Format

```markdown
# Tailored Resume: [Company] - [Role]
Date: [Date]

## Summary
[2-3 sentence summary highlighting relevant experience]

## Experience
### [Job Title] at [Company] | [Dates]
- [Relevant achievement 1]
- [Relevant achievement 2]

## Skills
- [Skill 1] - [Skill 2] - [Skill 3]

## Education
[Relevant education]
```

## Examples

- "Tailor my resume for the Google SWE position" → read job description, read master resume, create tailored version
- "What experience should I highlight for this ML role?" → search notes for ML-related projects
- "Update my master resume with new project" → read and update master resume
