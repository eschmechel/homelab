# Content Converter Skill

Convert web content, YouTube videos, and PDFs into Obsidian notes.

## When to Use

Use this skill when the user wants to:
- Convert a YouTube video to notes
- Save a webpage as notes
- Extract text from a PDF
- Create notes from online content

## How It Works

1. **YouTube**: Extract transcript/video info, create note with summary
2. **Web**: Fetch webpage content, clean and save as markdown
3. **PDF**: Extract text, save to note

## Tools

Use available tools to:
- `gh_grep` or web fetch - Get video/webpage content
- `obsidian-vault` - Write notes to vault
- Use external tools like yt-dlp, pdftotext

## YouTube Conversion

1. Get video ID from URL
2. Use yt-dlp to get transcript: `yt-dlp --write-subs --sub-lang en --skip-download URL`
3. Create note in `50-Converted/YouTube/` with:
   - Title
   - URL
   - Transcript
   - Key timestamps
   - Summary

## Web Conversion

1. Fetch webpage using curl or web fetch
2. Extract main content (remove nav, footer, etc.)
3. Convert to markdown
4. Save to `50-Converted/Web/`

## PDF Conversion

1. Extract text using pdftotext or similar
2. Create note in `50-Converted/PDFs/`
3. Add metadata (title, source, date)

## Note Structure

```markdown
# {{Title}}

**Source:** {{URL}}
**Date:** {{Date}}
**Type:** YouTube | Web | PDF

## Summary

## Key Points

## Transcript/Content

## Notes

## Related
```

## Vault Location

Save converted content to:
- `50-Converted/YouTube/`
- `50-Converted/Web/`
- `50-Converted/PDFs/`

## Examples

- "Convert this YouTube video to notes" → Extract transcript, save to vault
- "Save this webpage as notes" → Fetch and convert to markdown
- "Extract text from this PDF" → Convert PDF to note

## Requirements

Install on your system:
```bash
# YouTube transcripts
pip install yt-dlp

# PDF text extraction
sudo apt install poppler-utils  # pdftotext
```

## Notes

- Always include source URL
- Add tags for organization
- Include date of conversion
