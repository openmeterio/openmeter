import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { theme } from '@/lib/theme'

export default function NotFoundPage() {
  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <div className="max-w-md w-full text-center space-y-4">
        <h1 className={cn('text-4xl font-bold', theme.brand.title)}>Not found</h1>
        <p className="text-muted-foreground">
          This shared blueprint may have been removed, or the URL is incorrect.
        </p>
        <Link to="/" className="inline-block text-teal hover:underline">
          ← Back home
        </Link>
      </div>
    </div>
  )
}
