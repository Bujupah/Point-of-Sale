// Thin fetch wrapper. This is the ONLY place that talks to the backend —
// every screen goes through here, never straight to fetch(), so auth
// headers, error shape, and the base URL are handled in one spot.
//
// The session token is a disposable credential, not business data, so
// localStorage is an acceptable place for it (brief §5 only forbids storing
// authoritative business records — products, sales, cash, etc. — client
// side; a bearer token is neither).
const TOKEN_KEY = 'pos.session_token'

export function getToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

export function setToken(token: string | null) {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token)
    else localStorage.removeItem(TOKEN_KEY)
  } catch {
    // Storage unavailable (private mode, etc.) — session simply won't
    // survive a refresh. The app remains usable for the current session.
  }
}

export class ApiError extends Error {
  code: string
  status: number
  constructor(code: string, message: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (res.status === 204) return undefined as T

  const isJSON = res.headers.get('content-type')?.includes('application/json')
  const data = isJSON ? await res.json() : await res.text()

  if (!res.ok) {
    if (isJSON && data?.error) {
      throw new ApiError(data.error.code, data.error.message, res.status)
    }
    throw new ApiError('NETWORK_ERROR', typeof data === 'string' ? data : 'Request failed', res.status)
  }
  return data as T
}

export const api = {
  get: <T,>(path: string) => request<T>('GET', path),
  post: <T,>(path: string, body?: unknown) => request<T>('POST', path, body ?? {}),
  put: <T,>(path: string, body?: unknown) => request<T>('PUT', path, body ?? {}),
  del: <T,>(path: string) => request<T>('DELETE', path),
}

export function qs(params: Record<string, string | number | undefined | null>): string {
  const usp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') usp.set(k, String(v))
  }
  const s = usp.toString()
  return s ? `?${s}` : ''
}
