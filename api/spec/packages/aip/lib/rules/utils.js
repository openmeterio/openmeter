/**
 * Exceptions for PascalCase naming convention.
 */
const pascalCaseExceptions = ['OAuth2', 'URL', 'API', 'UI', 'ID']

/**
 * Checks whether a given value is in PascalCase
 * @param value the value to check
 * @returns true if the value is in PascalCase
 */
export function isPascalCaseNoAcronyms(value) {
  if (value === undefined || value === null || value === '') {
    return true
  }

  return new RegExp(
    `^(?:[A-Z][a-z0-9]+|${pascalCaseExceptions.join('|')})+[A-Z]?$|^[A-Z]+$`,
  ).test(value)
}

/**
 * Checks whether a given value is in camelCase
 * @param value the value to check
 * @returns true if the value is in camelCase
 */
export function isCamelCaseNoAcronyms(value) {
  if (value === undefined || value === null || value === '') {
    return true
  }

  return /^[^a-zA-Z0-9]?[a-z][a-z0-9]*([A-Z][a-z0-9]+)*[A-Z]?$/.test(value)
}

/**
 * Checks whether a given value is in snake_case
 * @param value the value to check
 * @returns true if the value is in snake_case
 */
export function isSnakeCase(value) {
  if (value === undefined || value === null || value === '') {
    return true
  }

  return /^([a-z0-9]+_)*[a-z0-9]+$/.test(value)
}

/**
 * Checks whether a given value is in kebab-case
 * @param value the value to check
 * @returns true if the value is in kebab-case
 */
export function isKebabCase(value) {
  if (value === undefined || value === null || value === '') {
    return true
  }

  return /^([a-z0-9]+(-[a-z0-9]+)*)$/.test(value)
}

/**
 * Detect the dominant line separator in a source string.
 * @param {string} source
 * @returns {'\n' | '\r\n'}
 */
export function detectNewline(source) {
  return source.includes('\r\n') ? '\r\n' : '\n'
}

/**
 * Return the indentation (whitespace from start of line) preceding `position`.
 * @param {string} source
 * @param {number} position
 */
export function getIndentBefore(source, position) {
  const lineStart = source.lastIndexOf('\n', position - 1) + 1
  const between = source.slice(lineStart, position)
  const match = between.match(/^[ \t]*/)
  return match ? match[0] : ''
}

/**
 * Strip `/** ... *\/` framing and per-line `*` decoration from a doc comment,
 * returning the inner Markdown body. Trims leading/trailing blank lines.
 * @param {string} raw the full comment text including `/** ... *\/`
 */
export function extractMarkdownFromDocComment(raw) {
  let body = raw
  if (body.startsWith('/**')) body = body.slice(3)
  if (body.endsWith('*/')) body = body.slice(0, -2)

  const lines = body.split(/\r?\n/).map((line) => {
    // Strip leading whitespace + a single `*` + an optional space.
    return line.replace(/^[ \t]*\*[ \t]?/, '').replace(/[ \t]+$/, '')
  })

  let start = 0
  let end = lines.length
  while (start < end && lines[start].trim() === '') start++
  while (end > start && lines[end - 1].trim() === '') end--
  return lines.slice(start, end).join('\n')
}

/**
 * Re-wrap a Markdown body as a TypeSpec doc comment, applying `indent` on each
 * line and using `newline` between lines.
 * @param {string} markdown
 * @param {string} indent
 * @param {'\n' | '\r\n'} newline
 */
export function wrapMarkdownAsDocComment(markdown, indent, newline) {
  const trimmed = markdown.replace(/\n+$/, '')
  if (trimmed === '') {
    return `/**${newline}${indent} */`
  }
  const lines = trimmed.split('\n')
  const out = [
    '/**',
    ...lines.map((line) =>
      line.length === 0 ? `${indent} *` : `${indent} * ${line}`,
    ),
    `${indent} */`,
  ]
  return out.join(newline)
}
