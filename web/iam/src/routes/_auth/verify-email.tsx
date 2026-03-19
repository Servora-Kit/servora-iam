import { createFileRoute, Link, useSearch } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { CheckCircle2, Loader2, XCircle } from 'lucide-react'
import { Button } from '#/components/ui/button'
import { iamClients } from '#/api'
import { useResendVerification } from '#/hooks/use-resend-verification'
import type { ApiError } from '@servora/web-pkg/request'
import { isKratosReason } from '@servora/web-pkg/errors'

export const Route = createFileRoute('/_auth/verify-email')({
  validateSearch: (search: Record<string, unknown>) => ({
    token: (search.token as string) || '',
    email: (search.email as string) || '',
  }),
  component: VerifyEmailPage,
})

type Status = 'verifying' | 'success' | 'expired' | 'error'

function VerifyEmailPage() {
  const { token, email } = useSearch({ from: '/_auth/verify-email' })
  const [status, setStatus] = useState<Status>('verifying')
  const verified = useRef(false)
  const { resend, resending, message: resendMsg } = useResendVerification(email)

  useEffect(() => {
    if (!token || verified.current) return
    verified.current = true

    iamClients.authn
      .VerifyEmail({ token })
      .then(() => setStatus('success'))
      .catch((err: unknown) => {
        const apiErr = err as ApiError
        setStatus(isKratosReason(apiErr, 'INVALID_VERIFICATION_TOKEN') || isKratosReason(apiErr, 'TOKEN_EXPIRED')
          ? 'expired'
          : 'error')
      })
  }, [token])

  return (
    <div className="flex flex-col items-center gap-6 text-center">
      {status === 'verifying' && (
        <>
          <div className="flex size-20 items-center justify-center rounded-full bg-muted">
            <Loader2 className="size-8 animate-spin text-primary" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">正在验证邮箱</h1>
            <p className="mt-2 text-muted-foreground">请稍候…</p>
          </div>
        </>
      )}

      {status === 'success' && (
        <>
          <div className="flex size-20 items-center justify-center rounded-full bg-success/15">
            <CheckCircle2 className="size-10 text-success" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">邮箱验证成功</h1>
            <p className="mt-2 text-muted-foreground">
              您的邮箱已验证，现在可以登录了。
            </p>
          </div>
          <Link
            to="/login"
            search={{ redirect: '', authRequestID: '' }}
            className="w-full"
          >
            <Button className="h-10 w-full">去登录</Button>
          </Link>
        </>
      )}

      {(status === 'expired' || status === 'error') && (
        <>
          <div className="flex size-20 items-center justify-center rounded-full bg-destructive/10">
            <XCircle className="size-10 text-destructive" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">
              {status === 'expired' ? '验证链接已过期' : '验证失败'}
            </h1>
            <p className="mt-2 text-muted-foreground">
              {status === 'expired'
                ? '该验证链接已过期（有效期 24 小时），请重新发送验证邮件。'
                : '验证链接无效，请检查邮件或重新发送。'}
            </p>
          </div>

          {resendMsg && (
            <p className="text-sm text-muted-foreground">{resendMsg}</p>
          )}

          <div className="flex w-full flex-col gap-3">
            {email && (
              <Button
                variant="outline"
                className="h-10 w-full"
                onClick={resend}
                disabled={resending}
              >
                {resending ? '发送中...' : '重新发送验证邮件'}
              </Button>
            )}
            <Link
              to="/login"
              search={{ redirect: '', authRequestID: '' }}
              className="w-full"
            >
              <Button variant="ghost" className="h-10 w-full">
                去登录
              </Button>
            </Link>
          </div>
        </>
      )}
    </div>
  )
}
