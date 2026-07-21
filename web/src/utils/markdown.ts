import { marked } from 'marked'
import DOMPurify from 'dompurify'

// marked.parse is synchronous by default, but its type union includes a Promise
// form. Pin the options so we always get a string, then sanitize for safe HTML.
marked.setOptions({ async: false, gfm: true, breaks: false })

export function renderMarkdown(content: string): string {
  if (!content) return ''
  const raw = marked.parse(content)
  const html = typeof raw === 'string' ? raw : ''
  return DOMPurify.sanitize(html)
}

// stripMarkdown returns a single-line plain-text excerpt of markdown content.
export function stripMarkdown(content: string, max = 140): string {
  const text = content
    .replace(/^---[\s\S]*?---\n?/, '') // drop frontmatter
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`[^`]*`/g, ' ')
    .replace(/[#>*_\-[\]()!]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  return text.length > max ? text.slice(0, max) + '…' : text
}

// tagsFromString / tagsFromString helpers: tags are stored comma-separated.
export function parseTags(tags: string): string[] {
  return tags.split(',').map(t => t.trim()).filter(Boolean)
}

export function joinTags(tags: string[]): string {
  return tags.join(', ')
}
