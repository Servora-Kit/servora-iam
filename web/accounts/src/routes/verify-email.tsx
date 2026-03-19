import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { z } from 'zod'
import { authnApi } from '#/api'
import { AuthCard } from '#/components/auth-card'

const searchSchema = z.object({
  token: z.string(),
})

export const Route = createFileRoute('/verify-email')({
  validateSearch: searchSchema,
  component: VerifyEmailPage,
})

type Status = 'pending' | 'success' | 'error'

function VerifyEmailPage() {
  const { token } = Route.useSearch()
  const [status, setStatus] = useState<Status>('pending')
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    authnApi
      .verifyEmail(token)
      .then(() => setStatus('success'))
      .catch((err: unknown) => {
        setStatus('error')
        setMessage(err instanceof Error ? err.message : 'Verification failed')
      })
  }, [token])

  return (
    <AuthCard title="Email Verification">
      {status === 'pending' && (
        <p className="text-sm text-[#4c4f69] dark:text-[#cdd6f4] text-center">
          Verifying your email…
        </p>
      )}
      {status === 'success' && (
        <p className="text-sm text-[#40a02b] dark:text-[#a6e3a1] text-center font-medium">
          Your email has been verified. You can now sign in.
        </p>
      )}
      {status === 'error' && (
        <p className="text-sm text-[#d20f39] dark:text-[#f38ba8] text-center">
          {message ?? 'Verification failed. The link may have expired.'}
        </p>
      )}
    </AuthCard>
  )
}
