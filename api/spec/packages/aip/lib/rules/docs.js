import {
  createRule,
  defineCodeFix,
  getDoc,
  getSourceLocation,
  paramMessage,
} from '@typespec/compiler'
import * as prettier from 'prettier'
import {
  detectNewline,
  extractMarkdownFromDocComment,
  getIndentBefore,
  wrapMarkdownAsDocComment,
} from './utils.js'

export const docDecoratorRule = createRule({
  name: 'doc-decorator',
  severity: 'warning',
  description: 'Ensure documentation.',
  messages: {
    default: paramMessage`Missing documentation for ${'name'} ${'type'}`,
  },
  create: (context) => ({
    model: (target) => {
      if (target.name && !getDoc(context.program, target)) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }

      if (target.name.endsWith('Response')) {
        return
      }

      for (const [name, property] of target.properties) {
        if (
          target.name &&
          name &&
          !['_', 'contentType'].includes(name) &&
          !getDoc(context.program, property)
        ) {
          context.reportDiagnostic({
            target: property,
            format: {
              name: `${target.name}.${name}`,
            },
          })
        }
      }
    },
    enum: (target) => {
      if (target.name && !getDoc(context.program, target)) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
    union: (target) => {
      if (target.name && !getDoc(context.program, target)) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
  }),
})

/**
 * Format a doc-comment Markdown body through Prettier.
 * Returns the formatted Markdown body (no `/** *\/` framing).
 * @param {string} markdown
 * @param {{ printWidth?: number, proseWrap?: 'always' | 'never' | 'preserve' }} [options]
 */
async function formatDocMarkdown(markdown, options = {}) {
  if (markdown.trim() === '') return ''
  return await prettier.format(markdown, {
    parser: 'markdown',
    printWidth: options.printWidth ?? 80,
    proseWrap: options.proseWrap ?? 'always',
  })
}

/**
 * Build a code fix that replaces a doc comment with a precomputed string.
 * The Prettier work happens before this is constructed; the fix callback is
 * sync and just emits the replacement.
 *
 * @param {import('@typespec/compiler').SourceLocation} location
 *   The full `/** ... *\/` source range.
 * @param {string} newText The replacement text, including `/**` and `*\/`.
 */
function createFormatDocCommentCodeFix(location, newText) {
  return defineCodeFix({
    id: 'format-doc-comment',
    label: 'Format doc comment',
    fix(context) {
      return context.replaceText(location, newText)
    },
  })
}

/**
 * Collect every `DocNode` reachable from the program by walking semantic
 * targets that can carry doc comments. We use the existing semantic listener
 * surface (model/property/enum/etc.) rather than a private AST walker.
 */
function collectDocNodes(target, sink) {
  const node = target.node
  if (!node || !node.docs || node.docs.length === 0) return
  for (const doc of node.docs) sink.push(doc)
}

export const docFormatRule = createRule({
  name: 'doc-format',
  severity: 'warning',
  description:
    'Format doc comment bodies as Markdown using Prettier (proseWrap=always).',
  messages: {
    default:
      'Doc comment is not formatted. Apply the suggested fix to reformat as Markdown.',
  },
  // Async because Prettier 3.x's `format` is async.
  async: true,
  create: (context) => {
    /** @type {import('@typespec/compiler').DocNode[]} */
    const docNodes = []

    const collect = (target) => collectDocNodes(target, docNodes)

    return {
      model: collect,
      modelProperty: collect,
      enum: collect,
      enumMember: collect,
      union: collect,
      unionVariant: collect,
      operation: collect,
      interface: collect,
      scalar: collect,
      namespace: collect,

      async exit() {
        // Deduplicate: a doc may be visited via multiple semantic kinds.
        const seen = new Set()
        const work = []
        for (const doc of docNodes) {
          if (seen.has(doc)) continue
          seen.add(doc)
          work.push(processDoc(doc, context))
        }
        await Promise.all(work)
      },
    }
  },
})

/**
 * Compute a formatted replacement for a single DocNode and, if it differs
 * from the source, report a diagnostic with an attached code fix.
 * @param {import('@typespec/compiler').DocNode} doc
 * @param {import('@typespec/compiler').LinterRuleContext<any>} context
 */
async function processDoc(doc, context) {
  const location = getSourceLocation(doc)
  const source = location.file.text
  const raw = source.slice(location.pos, location.end)

  // Defensive: only format actual `/** ... */` blocks.
  if (!raw.startsWith('/**') || !raw.endsWith('*/')) return

  const indent = getIndentBefore(source, location.pos)
  const newline = detectNewline(source)

  let markdown
  try {
    markdown = extractMarkdownFromDocComment(raw)
  } catch {
    return
  }

  let formatted
  try {
    formatted = await formatDocMarkdown(markdown)
  } catch {
    // If Prettier can't parse the body, leave it alone.
    return
  }

  const replacement = wrapMarkdownAsDocComment(formatted, indent, newline)
  if (replacement === raw) return

  context.reportDiagnostic({
    target: doc,
    codefixes: [createFormatDocCommentCodeFix(location, replacement)],
  })
}
