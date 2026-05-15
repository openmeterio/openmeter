import { useEffect } from 'react'

interface Props {
  message: string | null
  onDismiss: () => void
}

export default function Toast({ message, onDismiss }: Props) {
  useEffect(() => {
    if (!message) return
    const id = setTimeout(onDismiss, 2000)
    return () => clearTimeout(id)
  }, [message, onDismiss])

  if (!message) return null
  return (
    <div className="fixed bottom-6 right-6 bg-ink text-papaya-50 px-5 py-3 rounded-xl shadow-2xl shadow-ink/30 ring-1 ring-ink/10 z-50 text-sm font-medium animate-in fade-in slide-in-from-bottom-2 duration-300">
      {message}
    </div>
  )
}
