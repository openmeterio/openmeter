import { useState } from 'react'
import { Copy, Check, ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import { theme } from '@/lib/theme'
import { GhostLogo } from '@/components/GhostLogo'

const INSTALL_CMD = 'npx @bitraptors/archie /path/to/your/project'

export default function HomePage() {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    await navigator.clipboard.writeText(INSTALL_CMD)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center p-8">
      <div className="max-w-2xl w-full text-center space-y-6">
        <GhostLogo size={72} className="mx-auto" />
        <h1 className={cn('text-5xl font-bold', theme.brand.title)}>Archie</h1>
        <p className="text-xl text-muted-foreground">
          Senior-architect-level analysis of any codebase — shareable via URL.
        </p>

        <div className="pt-4 space-y-3">
          <p className="text-sm text-muted-foreground">Try it on your codebase:</p>
          <div
            className={cn(
              'rounded-lg p-4 font-mono text-sm inline-flex items-center gap-3',
              theme.console.bg,
              theme.console.text
            )}
          >
            <code>{INSTALL_CMD}</code>
            <button onClick={copy} className="hover:opacity-80 transition-opacity" title="Copy">
              {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
            </button>
          </div>
        </div>

        <div className="pt-8">
          <a
            href="https://github.com/BitRaptors/Archie"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-teal"
          >
            Learn more on GitHub <ExternalLink className="w-3 h-3" />
          </a>
        </div>
      </div>
    </div>
  )
}
