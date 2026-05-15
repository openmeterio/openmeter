import { useEffect, useState, useRef } from 'react'
import MarkdownPane from './MarkdownPane'
import TreeNav from './TreeNav'
import IntentLayerEmptyState from './IntentLayerEmptyState'

export default function FolderClaudeMdsBrowser() {
  const [status, setStatus] = useState<{ exists: boolean; count: number } | null>(null)
  const [files, setFiles] = useState<Record<string, string> | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [sidebarWidth, setSidebarWidth] = useState(220)
  const isResizing = useRef(false)

  useEffect(() => {
    fetch('/api/intent-layer-status')
      .then((r) => {
        if (!r.ok) {
          throw new Error(
            r.status === 404
              ? 'This Archie viewer is out of date — /api/intent-layer-status missing. Stop the viewer.py process and relaunch it to pick up the new endpoints.'
              : `intent-layer-status HTTP ${r.status}`,
          )
        }
        return r.json()
      })
      .then((s: { exists: boolean; count: number }) => {
        setStatus(s)
        if (s.exists) {
          return fetch('/api/folder-claude-mds')
            .then((r) => {
              if (!r.ok) throw new Error(`folder-claude-mds HTTP ${r.status}`)
              return r.json()
            })
            .then((data: Record<string, string>) => {
              setFiles(data)
              const firstFile = Object.keys(data)[0]
              if (firstFile) {
                const folder = firstFile.split('/').slice(0, -1).join('/') || '.'
                setSelected(folder)
              }
            })
        }
      })
      .catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing.current) return
      const newWidth = e.clientX - 312
      if (newWidth > 160 && newWidth < 600) {
        setSidebarWidth(newWidth)
      }
    }
    const handleMouseUp = () => {
      if (isResizing.current) {
        isResizing.current = false
        document.body.style.cursor = 'default'
        document.body.style.userSelect = 'auto'
      }
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [])

  if (error)
    return (
      <div className="bg-white border border-papaya-300/40 rounded-2xl p-8 shadow-sm text-red-600">
        Failed to load: {error}
      </div>
    )
  if (!status)
    return (
      <div className="bg-white border border-papaya-300/40 rounded-2xl p-8 shadow-sm text-ink/60">
        Loading…
      </div>
    )
  if (!status.exists) return <IntentLayerEmptyState count={status.count} />
  if (!files)
    return (
      <div className="bg-white border border-papaya-300/40 rounded-2xl p-8 shadow-sm text-ink/60">
        Loading folders…
      </div>
    )

  return (
    <div className="flex flex-col lg:flex-row h-[calc(100vh-8rem)]">
      <aside
        className="shrink-0 overflow-y-auto pr-3 custom-scrollbar"
        style={{ width: `${sidebarWidth}px` }}
      >
        <p className="text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4 px-2">
          Folders
        </p>
        <TreeNav paths={Object.keys(files)} selected={selected} onSelect={setSelected} />
      </aside>

      {/* Resize Handle */}
      <div
        className="hidden lg:block w-1 group cursor-col-resize relative z-10"
        onMouseDown={(e) => {
          e.preventDefault()
          isResizing.current = true
          document.body.style.cursor = 'col-resize'
          document.body.style.userSelect = 'none'
        }}
      >
        <div className="absolute inset-y-0 -left-1 -right-1 group-hover:bg-teal/10 transition-colors" />
        <div className="absolute inset-y-8 left-0 w-[1px] bg-papaya-300 opacity-50 group-hover:bg-teal group-hover:opacity-100 transition-all" />
      </div>

      <main className="flex-1 overflow-y-auto bg-white/60 backdrop-blur-xl border border-white/80 rounded-[32px] p-8 lg:p-16 shadow-2xl shadow-ink/5 custom-scrollbar relative ml-3">
        <div className="absolute inset-0 bg-gradient-to-br from-white/40 to-transparent rounded-[32px] pointer-events-none" />
        <div className="relative">
          {selected && files && (
            <MarkdownPane 
              files={Object.entries(files)
                .filter(([path]) => (path.split('/').slice(0, -1).join('/') || '.') === selected)
                .map(([path, content]) => ({ filename: path.split('/').pop()!, content }))
              } 
            />
          )}
        </div>
      </main>
    </div>
  )
}
