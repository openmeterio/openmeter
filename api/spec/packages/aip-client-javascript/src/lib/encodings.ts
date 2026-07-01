export function encodePath(
  template: string,
  params: Record<string, string | number>,
): string {
  return template.replace(/\{(\w+)\}/g, (_, key: string) => {
    const value = params[key]
    if (value === undefined) {
      throw new Error(`missing path parameter: ${key}`)
    }
    return encodeURIComponent(String(value))
  })
}

function serializeDeepObject(
  prefix: string,
  value: unknown,
  parts: Array<[string, string]>,
): void {
  if (value === null || value === undefined) {
    return
  }
  if (Array.isArray(value)) {
    parts.push([prefix, value.map(String).join(',')])
  } else if (typeof value === 'object') {
    for (const [k, v] of Object.entries(value)) {
      serializeDeepObject(`${prefix}[${k}]`, v, parts)
    }
  } else {
    parts.push([prefix, String(value)])
  }
}

function serializeParams(
  params: Record<string, unknown>,
): Array<[string, string]> {
  const parts: Array<[string, string]> = []
  for (const [key, value] of Object.entries(params)) {
    serializeDeepObject(key, value, parts)
  }
  return parts
}

export function querySerializer(params: Record<string, unknown>): string {
  const parts = serializeParams(params).map(
    ([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`,
  )
  return parts.length ? `?${parts.join('&')}` : ''
}

export function toURLSearchParams(
  params: Record<string, unknown>,
): URLSearchParams {
  const search = new URLSearchParams()
  for (const [key, value] of serializeParams(params)) {
    search.append(key, value)
  }
  return search
}

export function encodeSort(
  sort: { by?: string; order?: 'asc' | 'desc' } | undefined,
  encodeField: (field: string) => string = (field) => field,
): string | undefined {
  if (!sort?.by) {
    return undefined
  }
  const by = encodeField(sort.by)
  if (sort.order === 'desc') {
    return `${by} desc`
  }
  if (sort.order === 'asc') {
    return `${by} asc`
  }
  return by
}
