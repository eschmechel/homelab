# Job Tracker Skill

Track and manage job applications, interviews, and career progress in your Obsidian vault.

## When to Use

Use this skill when the user wants to:
- Track job applications
- Log interview stages and notes
- Manage job search progress
- Get overview of their job search status

## Vault Structure

Your job tracking files should be in:
- `20-Career/` - Main career folder
- `60-Internship-Tracker/` - Internship specific tracking

## Tools

Use the obsidian-vault MCP to:
- `search_content` - Search for job-related notes
- `search_files` - Find job application files
- `read_file` - Read a specific job application
- `write_file` - Create/update job application notes
- `list_directory` - List job folders
- `create_directory` - Create new job folders

## Job Application Format

Create notes for each job in this format:

```markdown
# Job: [Company Name] - [Position]
Status: [Applied/Interviewing/Offered/Rejected]
Date Applied: [Date]
Source: [Where you found the job]
Salary: [$X - $Y]
Location: [Remote/Hybrid/On-site]
Notes:

## Interview Stages
- [ ] Application Submitted
- [ ] Initial Screen
- [ ] Technical Interview
- [ ] Final Interview
- [ ] Offer Received

## Notes
- 
```

## How to Use

1. **Add new job**: Create a new note in 20-Career with company and position
2. **Update status**: Edit the Status field
3. **Log interview**: Add notes to the Notes section
4. **Search jobs**: Use search_content to find all jobs at a company
5. **List all jobs**: Use search_files to find all .md files in Career folder

## Examples

- "I applied to a new job at Google" → create new note in 20-Career
- "What's my job search progress?" → list all job notes
- "Update my Amazon application to final round" → read and update the note
- "Show me all jobs I've applied to" → search_content for "Status: Applied"
