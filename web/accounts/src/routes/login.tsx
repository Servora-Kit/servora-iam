import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { z } from 'zod'
import { authnApi } from '#/api'
import { AuthCard } from '#/components/auth-card'

const searchSchema = z.object({
  authRequestID: z.string(),
})

export const Route = createFileRoute('/login')({
  validateSearch: searchSchema,
  component: LoginPage,
})

function LoginPage() {
  const { authRequestID } = Route.useSearch()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const result = await authnApi.loginComplete(authRequestID, email, password)
      window.location.href = result.callbackURL
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthCard title="Sign In">
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <p className="text-sm text-[#d20f39] dark:text-[#f38ba8] text-center">{error}</p>
        )}
        <div className="space-y-1.5">
          <label
            htmlFor="email"
            className="block text-sm font-medium text-[#4c4f69] dark:text-[#cdd6f4]"
          >
            Email
          </label>
          <input
            id="email"
            type="email"
            required
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-[#ccd0da] dark:border-[#313244] bg-[#eff1f5] dark:bg-[#11111b] text-[#4c4f69] dark:text-[#cdd6f4] focus:outline-none focus:ring-2 focus:ring-[#1e66f5] dark:focus:ring-[#89b4fa]"
          />
        </div>
        <div className="space-y-1.5">
          <label
            htmlFor="password"
            className="block text-sm font-medium text-[#4c4f69] dark:text-[#cdd6f4]"
          >
            Password
          </label>
          <input
            id="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-[#ccd0da] dark:border-[#313244] bg-[#eff1f5] dark:bg-[#11111b] text-[#4c4f69] dark:text-[#cdd6f4] focus:outline-none focus:ring-2 focus:ring-[#1e66f5] dark:focus:ring-[#89b4fa]"
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          className="w-full py-2.5 text-sm font-medium rounded-lg bg-[#1e66f5] dark:bg-[#89b4fa] text-white dark:text-[#1e1e2e] hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {loading ? 'Signing in…' : 'Sign In'}
        </button>
      </form>
    </AuthCard>
  )
}
