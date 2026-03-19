import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { authnApi } from '#/api'
import { AuthCard } from '#/components/auth-card'

export const Route = createFileRoute('/register')({
  component: RegisterPage,
})

function RegisterPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirm, setPasswordConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (password !== passwordConfirm) {
      setError('Passwords do not match')
      return
    }
    setError(null)
    setLoading(true)
    try {
      await authnApi.signup({ name, email, password, passwordConfirm })
      setSuccess(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <AuthCard title="Check Your Email">
        <p className="text-sm text-[#4c4f69] dark:text-[#cdd6f4] text-center">
          We&apos;ve sent a verification link to <strong>{email}</strong>. Please check your inbox and
          click the link to activate your account.
        </p>
      </AuthCard>
    )
  }

  return (
    <AuthCard title="Create Account">
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <p className="text-sm text-[#d20f39] dark:text-[#f38ba8] text-center">{error}</p>
        )}
        <div className="space-y-1.5">
          <label
            htmlFor="name"
            className="block text-sm font-medium text-[#4c4f69] dark:text-[#cdd6f4]"
          >
            Username
          </label>
          <input
            id="name"
            type="text"
            required
            autoFocus
            minLength={5}
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-[#ccd0da] dark:border-[#313244] bg-[#eff1f5] dark:bg-[#11111b] text-[#4c4f69] dark:text-[#cdd6f4] focus:outline-none focus:ring-2 focus:ring-[#1e66f5] dark:focus:ring-[#89b4fa]"
          />
        </div>
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
        <div className="space-y-1.5">
          <label
            htmlFor="passwordConfirm"
            className="block text-sm font-medium text-[#4c4f69] dark:text-[#cdd6f4]"
          >
            Confirm Password
          </label>
          <input
            id="passwordConfirm"
            type="password"
            required
            value={passwordConfirm}
            onChange={(e) => setPasswordConfirm(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-[#ccd0da] dark:border-[#313244] bg-[#eff1f5] dark:bg-[#11111b] text-[#4c4f69] dark:text-[#cdd6f4] focus:outline-none focus:ring-2 focus:ring-[#1e66f5] dark:focus:ring-[#89b4fa]"
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          className="w-full py-2.5 text-sm font-medium rounded-lg bg-[#1e66f5] dark:bg-[#89b4fa] text-white dark:text-[#1e1e2e] hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {loading ? 'Creating account…' : 'Create Account'}
        </button>
      </form>
    </AuthCard>
  )
}
