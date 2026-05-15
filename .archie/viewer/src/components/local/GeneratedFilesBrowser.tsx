import { useEffect, useState, useRef } from 'react'
import MarkdownPane from './MarkdownPane'
import TreeNav from './TreeNav'

export default function GeneratedFilesBrowser() {
  const [files, setFiles] = useState<Record<string, string> | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [sidebarWidth, setSidebarWidth] = useState(220)
  const isResizing = useRef(false)

  useEffect(() => {
    fetch('/api/generated-files')
      .then((r) => {
        if (!r.ok) {
          throw new Error(
            r.status === 404
              ? 'This Archie viewer is out of date — /api/generated-files missing. Stop the viewer.py process and relaunch it to pick up the new endpoints.'
              : `HTTP ${r.status}`,
          )
        }
        return r.json()
      })
      .then((data) => {
        setFiles(data)
        // File-mode tree: select the first file directly so the main pane
        // shows content immediately. The user wanted "see all files in the
        // structure, click a file to see its content" — there's no "folder
        // view" intermediate step to land on.
        const firstFile = Object.keys(data)[0]
        if (firstFile) setSelected(firstFile)
      })
      .catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing.current) return
      // Calculate width based on the movement relative to the sidebar's start
      // Since it's nested in ReportPage's main area (ml-72), we can just track delta 
      // or use a simpler clientX based calculation if we assume fixed layout.
      // Better: use the mouse position to set the width directly.
      const newWidth = e.clientX - 312 // adjustment for the Archie sidebar (w-72) + padding
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
        Failed to load generated files: {error}
      </div>
    )
  if (!files)
    return (
      <div className="bg-white border border-papaya-300/40 rounded-2xl p-8 shadow-sm text-ink/60">
        Loading…
      </div>
    )
  if (Object.keys(files).length === 0)
    return (
      <div className="bg-white border border-papaya-300/40 rounded-2xl p-8 shadow-sm text-ink/70">
        No generated files yet — run{' '}
        <code className="bg-papaya-100 text-teal-700 px-1.5 py-0.5 rounded font-semibold">
          /archie-scan
        </code>{' '}
        first.
      </div>
    )

  return (
    <div className="flex flex-col lg:flex-row h-[calc(100vh-8rem)]">
      <aside 
        className="shrink-0 overflow-y-auto pr-6 custom-scrollbar"
        style={{ width: `${sidebarWidth}px` }}
      >
        <p className="text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4 px-2">
          Files
        </p>
        <TreeNav paths={Object.keys(files)} selected={selected} onSelect={setSelected} mode="files" />
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
          {selected && files && files[selected] !== undefined && (
            <MarkdownPane
              files={[{ filename: selected, content: files[selected] }]}
            />
          )}
        </div>
      </main>
    </div>
  )
}
