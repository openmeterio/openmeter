import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import 'highlight.js/styles/atom-one-dark.min.css'

interface FileContent {
  filename: string
  content: string
}

interface Props {
  files: FileContent[]
}

// Shared markdown surface for the local viewer's tab panes (Generated Files,
// Folder CLAUDE.mds). Mirrors the plugin set used in ReportPage's executive
// summary block (remark-gfm + rehype-highlight) so we don't drift into two
// divergent markdown configs. Light-palette `prose` styling so the panes feel
// native next to the cream + ink + teal blueprint shell.
export default function MarkdownPane({ files }: Props) {
  if (files.length === 0) return null

  return (
    <div className="space-y-24">
      {files.map((file, idx) => (
        <div key={file.filename} className="animate-in fade-in slide-in-from-bottom-4 duration-700" style={{ animationDelay: `${idx * 100}ms` }}>
          <div className="flex items-center gap-3 mb-10 text-ink/30 font-black uppercase tracking-[0.3em] text-[10px]">
             <span className="w-8 h-px bg-current" />
             <span>{file.filename}</span>
          </div>
          <div className="prose max-w-none lg:prose-lg prose-headings:text-ink prose-headings:font-black prose-headings:tracking-tight prose-h1:text-4xl prose-h2:text-2xl prose-p:text-ink/80 prose-p:leading-relaxed prose-li:text-ink/80 prose-strong:text-ink prose-code:bg-papaya-100 prose-code:text-teal-700 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:before:hidden prose-code:after:hidden prose-a:text-teal-700 prose-a:no-underline hover:prose-a:underline prose-blockquote:border-teal-500 prose-blockquote:text-ink/70 prose-blockquote:bg-teal-500/5 prose-blockquote:py-2 prose-blockquote:px-6 prose-blockquote:rounded-r-xl prose-blockquote:not-italic">
            <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
              {file.content}
            </ReactMarkdown>
          </div>
          {idx < files.length - 1 && (
            <div className="mt-24 h-px bg-gradient-to-r from-transparent via-papaya-300 to-transparent opacity-50" />
          )}
        </div>
      ))}
    </div>
  )
}
