/**
 * Accounts app API — authn-only, no user session store.
 * Uses native fetch with the Vite dev proxy (or VITE_API_BASE_URL in prod).
 */

import { env } from '#/env'

const base = env.VITE_API_BASE_URL

async function post<T>(path: string, body: unknown): Promise<T> {
  const resp = await fetch(`${base}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!resp.ok) {
    let message = `HTTP ${resp.status}`
    try {
      const data = (await resp.json()) as { message?: string }
      if (data.message) message = data.message
    } catch {
      // ignore parse errors
    }
    throw new Error(message)
  }

  return resp.json() as Promise<T>
}

export type SignupResult = { id?: string; name?: string; email?: string }
export type VerifyEmailResult = { success?: boolean }
export type RequestPasswordResetResult = { success?: boolean }
export type ResetPasswordResult = { success?: boolean }

export const authnApi = {
  signup(params: { name: string; email: string; password: string; passwordConfirm: string }) {
    return post<SignupResult>('/v1/auth/signup/using-email', params)
  },
  verifyEmail(token: string) {
    return post<VerifyEmailResult>('/v1/auth/verify-email', { token })
  },
  requestPasswordReset(email: string) {
    return post<RequestPasswordResetResult>('/v1/auth/request-password-reset', { email })
  },
  resetPassword(token: string, newPassword: string) {
    return post<ResetPasswordResult>('/v1/auth/reset-password', { token, newPassword })
  },
  loginComplete(authRequestID: string, email: string, password: string) {
    return post<{ callbackURL: string }>('/login/complete', { authRequestID, email, password })
  },
}
