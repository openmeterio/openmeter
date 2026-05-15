const SUPABASE_FUNCTIONS_URL =
  import.meta.env.VITE_SUPABASE_FUNCTIONS_URL ||
  'https://chlmyhkjnirrcrjdsvrc.supabase.co/functions/v1'

const ENTERPRISE_TOKEN = 'ext'

export interface SemanticDuplication {
  function?: string
  locations?: string[]
  recommendation?: string
}

export interface Bundle {
  blueprint: any
  health?: any
  scan_meta?: any
  rules_adopted?: any
  rules_proposed?: any
  scan_report?: string
  semantic_duplications?: SemanticDuplication[]
  findings?: any[]
}

export interface ReportResponse {
  bundle: Bundle
  created_at: string
}

function base64UrlDecode(s: string): string {
  const pad = s.length % 4 === 0 ? 0 : 4 - (s.length % 4)
  const b64 = s.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat(pad)
  return atob(b64)
}

async function fetchEnterpriseReport(): Promise<ReportResponse> {
  // Read the GET URL from the URL fragment. Fragments are never transmitted to
  // any server (including Vercel) — they live only in the user's browser.
  const fragment = window.location.hash.slice(1)
  if (!fragment) {
    throw new Error(
      'Enterprise share URL is missing its data fragment. Ask the owner to re-share.'
    )
  }

  let getUrl: string
  try {
    getUrl = base64UrlDecode(fragment)
  } catch {
    throw new Error('Enterprise share URL fragment is malformed.')
  }

  let res: Response
  try {
    res = await fetch(getUrl)
  } catch (e) {
    throw new Error(
      `Could not fetch from the customer bucket. This is usually a CORS ` +
        `misconfiguration (the bucket must allow ${window.location.origin}) ` +
        `or an expired presigned URL. Original error: ${e}`
    )
  }

  if (!res.ok) {
    if (res.status === 403) {
      throw new Error(
        'The share URL has expired or is unauthorized (HTTP 403). Ask the ' +
          'owner to re-run /archie-share.'
      )
    }
    throw new Error(`Fetch failed from customer bucket (HTTP ${res.status}).`)
  }

  return res.json()
}

export async function fetchReport(token: string): Promise<ReportResponse> {
  if (token === ENTERPRISE_TOKEN) {
    return fetchEnterpriseReport()
  }
  const res = await fetch(`${SUPABASE_FUNCTIONS_URL}/blueprint?token=${token}`)
  if (!res.ok) throw new Error(`Report not found (${res.status})`)
  return res.json()
}
