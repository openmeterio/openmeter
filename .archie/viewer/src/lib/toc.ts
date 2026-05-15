/** Shared ID contract — used by both parseNavigation() and ReactMarkdown heading components. */
export const generateId = (text: string): string =>
    text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')

/** Recursively extract plain text from React children (handles bold, links, code, etc.) */
export function extractText(children: any): string {
    if (children == null) return ''
    if (typeof children === 'string') return children
    if (typeof children === 'number') return String(children)
    if (Array.isArray(children)) return children.map(extractText).join('')
    if (children?.props?.children != null) return extractText(children.props.children)
    return ''
}

export interface NavSection {
    title: string
    id: string
    items: { title: string; id: string }[]
}

/** Parse raw markdown to extract h2/h3 heading tree for sidebar navigation. */
export function parseNavigation(content: string): NavSection[] {
    const lines = content.split('\n')
    const sections: NavSection[] = []
    let current: NavSection | null = null
    let inCodeBlock = false

    for (const line of lines) {
        if (line.trim().startsWith('```')) { inCodeBlock = !inCodeBlock; continue }
        if (inCodeBlock) continue

        if (line.match(/^##\s+/) && !line.match(/^###\s+/)) {
            const raw = line.replace(/^##\s+/, '').trim()
            const title = raw.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1').replace(/(\*\*|__|_|\*|`)/g, '')
            current = { title, id: generateId(title), items: [] }
            sections.push(current)
        } else if (line.match(/^###\s+/) && current) {
            const raw = line.replace(/^###\s+/, '').trim()
            const title = raw.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1').replace(/(\*\*|__|_|\*|`)/g, '')
            current.items.push({ title, id: generateId(title) })
        }
    }
    return sections
}
/** Extract only specific h2 sections from a markdown string. */
export function filterReportSections(content: string, requestedTitles: string[]): string {
    const lines = content.split('\n')
    const result: string[] = []
    let capturing = false
    let inCodeBlock = false

    const normalizedRequested = requestedTitles.map(t => t.toLowerCase())

    for (const line of lines) {
        if (line.trim().startsWith('```')) { inCodeBlock = !inCodeBlock }
        
        if (!inCodeBlock && line.match(/^##\s+/)) {
            const rawTitle = line.replace(/^##\s+/, '').trim().toLowerCase()
            capturing = normalizedRequested.some(req => rawTitle.includes(req))
        }

        if (capturing) {
            result.push(line)
        }
    }
    return result.join('\n')
}
