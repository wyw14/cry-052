export type ApiError = { code: string; message: string; fields?: Array<{field: string; message: string}>; request_id: string }
export type Page<T> = { items: T[]; page: number; size: number; total: number }

export const sessionToken = import.meta.env.VITE_LOCAL_SESSION_TOKEN || 'local-admin-session'
export const reviewerSessionToken = import.meta.env.VITE_REVIEWER_SESSION_TOKEN || 'local-reviewer-session'
const headers = { 'Content-Type': 'application/json', Authorization: `Bearer ${sessionToken}` }

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...init, headers: { ...headers, ...init.headers } })
  if (!response.ok) throw await response.json() as ApiError
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}
