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

## Workflow: Creating a Tailored Resume

1. **Create job application note**: In `21-Applications/2026-Spring/`
2. **Create resume folder**: In `22-Resumes/{Company}-{Role}/`
3. **Generate tailored resume**: Using the master resume, tailor for the job
4. **Save to folder**: Save as `resume.pdf` in the job's folder
5. **Link in application**: Update job note with link to resume

## Resume Structure

```
22-Resumes/
├── resume.tex          # Master LaTeX
├── resume.pdf         # Master PDF
├── Master-Index.md    # List of all resumes
├── Generated/         # AI-generated variations
├── ada_DevOps/       # Job-specific resumes
│   └── resume.pdf
├── Apple_Expert/
│   └── resume.pdf
└── ...
```

## Job Application Note Format

Each job application note should link to its resume:

```markdown
# Job Application: Company - Role

## Resume Used
[[../22-Resumes/company-role/resume.pdf]]
```

## Examples

- "Tailor my resume for the Google SWE position" → 
  1. Create job note in 21-Applications
  2. Create folder in 22-Resumes/Google-SWE/
  3. Generate tailored resume
  4. Link resume in job note
