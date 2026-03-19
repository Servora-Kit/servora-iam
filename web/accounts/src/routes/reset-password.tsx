import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { z } from 'zod'
import { authnApi } from '#/api'
import { AuthCard } from '#/components/auth-card'

const searchSchema = z.object({
  token: z.string().optional(),
})

export const Route = createFileRoute('/reset-password')({
  validateSearch: searchSchema,
  component: ResetPasswordPage,
})

function ResetPasswordPage() {
  const { token } = Route.useSearch()

  return token ? <ExecuteReset token={token} /> : <RequestReset />
}

function RequestReset() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await authnApi.requestPasswordReset(email)
      setSuccess(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <AuthCard title="Check Your Email">
        <p className="text-sm text-[#4c4f69] dark:text-[#cdd6f4] text-center">
          If an account exists for <strong>{email}</strong>, we&apos;ve sent a password reset link.
          Please check your inbox.
        </p>
      </AuthCard>
    )
  }

  return (
    <AuthCard title="Reset Password">
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
        <button
          type="submit"
          disabled={loading}
          className="w-full py-2.5 text-sm font-medium rounded-lg bg-[#1e66f5] dark:bg-[#89b4fa] text-white dark:text-[#1e1e2e] hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {loading ? 'Sending…' : 'Send Reset Link'}
        </button>
      </form>
    </AuthCard>
  )
}

function ExecuteReset({ token }: { token: string }) {
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match')
      return
    }
    setError(null)
    setLoading(true)
    try {
      await authnApi.resetPassword(token, newPassword)
      setSuccess(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Reset failed')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <AuthCard title="Password Updated">
        <p className="text-sm text-[#40a02b] dark:text-[#a6e3a1] text-center font-medium">
          Your password has been reset. You can now sign in with your new password.
        </p>
      </AuthCard>
    )
  }

  return (
    <AuthCard title="Set New Password">
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <p className="text-sm text-[#d20f39] dark:text-[#f38ba8] text-center">{error}</p>
        )}
        <div className="space-y-1.5">
          <label
            htmlFor="newPassword"
            className="block text-sm font-medium text-[#4c4f69] dark:text-[#cdd6f4]"
          >
            New Password
          </label>
          <input
            id="newPassword"
            type="password"
            required
            autoFocus
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-[#ccd0da] dark:border-[#313244] bg-[#eff1f5] dark:bg-[#11111b] text-[#4c4f69] dark:text-[#cdd6f4] focus:outline-none focus:ring-2 focus:ring-[#1e66f5] dark:focus:ring-[#89b4fa]"
          />
        </div>
        <div className="space-y-1.5">
          <label
            htmlFor="confirmPassword"
            className="block text-sm font-medium text-[#4c4f69] dark:text-[#cdd6f4]"
          >
            Confirm Password
          </label>
          <input
            id="confirmPassword"
            type="password"
            required
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-[#ccd0da] dark:border-[#313244] bg-[#eff1f5] dark:bg-[#11111b] text-[#4c4f69] dark:text-[#cdd6f4] focus:outline-none focus:ring-2 focus:ring-[#1e66f5] dark:focus:ring-[#89b4fa]"
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          className="w-full py-2.5 text-sm font-medium rounded-lg bg-[#1e66f5] dark:bg-[#89b4fa] text-white dark:text-[#1e1e2e] hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {loading ? 'Updating…' : 'Update Password'}
        </button>
      </form>
    </AuthCard>
  )
}
