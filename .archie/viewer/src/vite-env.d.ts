/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SUPABASE_FUNCTIONS_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
