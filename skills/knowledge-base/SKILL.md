# Knowledge Base Skill

Search and query your personal knowledge base stored in Obsidian.

## When to Use

Use this skill when the user wants to:
- Find information in their notes
- Search for a specific topic
- Query their knowledge base
- Read a specific note
- Get context from their personal wiki

## Vault Structure

Your Obsidian vault is located at `/opt/obsidian-vault` with these main folders:
- `00-Inbox` - New notes awaiting processing
- `10-Academic` - Academic notes and coursework
- `20-Career` - Career-related notes
  - `21-Applications/` - Job applications (2026-Spring/, Archive/)
  - `22-Resumes/` - Master resume and generated versions
  - `23-LinkedIn/` - LinkedIn contacts and strategies
  - `24-Interview-Prep/` - Interview preparation notes
  - `25-Company-Research/` - Company research and notes
  - `26-Salary-Tracker/` - Salary tracking and negotiation notes
- `30-Daily` - Daily journal entries
- `40-Knowledge` - Main knowledge base (wiki-style notes)
- `50-Converted` - Imported/converted notes
- `60-Internship-Tracker` - Internship applications and tracking
- `70-Productivity` - Productivity notes and systems
- `80-Reviews` - Review notes (books, articles, etc.)
- `90-Health` - Health and fitness notes
- `91-Finance` - Finance and budgeting notes
- `92-Lifestyle` - Lifestyle and personal development
- `99-Templates` - Note templates

## Tools

Use the obsidian-vault MCP to:
- `search_content` - Search for text across all notes
- `search_files` - Find files by name pattern
- `read_file` - Read a specific note
- `list_directory` - List contents of a folder
- `write_file` - Create or update a note
- `create_directory` - Create a new folder

## How to Search

1. **Find by content**: Use `search_content` with keywords
2. **Find by name**: Use `search_files` with glob patterns like `**/*.md`
3. **Read a note**: Use `read_file` with the full path (relative to vault root)
4. **Browse a folder**: Use `list_directory` to see what's in a folder

## Examples

- "Search my notes for anything about X" → use `search_content`
- "Do I have notes about Y?" → use `search_content`
- "What's in my 40-Knowledge folder?" → use `list_directory` for `40-Knowledge`
- "Read my note about Z" → use `read_file` for the specific file
- "Create a new note about X" → use `write_file` to create in appropriate folder

## Important Notes

- All paths are relative to the vault root (e.g., `40-Knowledge/topics X.md`)
- The MCP automatically prepends the vault path
- Search results show matching file paths and content snippets
- You can create new notes in any folder
