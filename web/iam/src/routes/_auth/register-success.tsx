import { createFileRoute, Link, useSearch } from '@tanstack/react-router'
import { Mail } from 'lucide-react'
import { Button } from '#/components/ui/button'
import { useResendVerification } from '#/hooks/use-resend-verification'

export const Route = createFileRoute('/_auth/register-success')({
  validateSearch: (search: Record<string, unknown>) => ({
    email: (search.email as string) || '',
  }),
  component: RegisterSuccessPage,
})

function RegisterSuccessPage() {
  const { email } = useSearch({ from: '/_auth/register-success' })
  const { resend, resending, message: resendMsg } = useResendVerification(email)

  return (
    <div className="flex flex-col items-center gap-6 text-center">
      <div className="flex size-20 items-center justify-center rounded-full bg-primary/10">
        <Mail className="size-10 text-primary" />
      </div>

      <div>
        <h1 className="text-2xl font-bold text-foreground">请验证您的邮箱</h1>
        <p className="mt-2 text-muted-foreground">
          我们已向{' '}
          <span className="font-medium text-foreground">
            {email || '您的邮箱'}
          </span>{' '}
          发送了验证邮件，请点击邮件中的链接完成注册。
        </p>
      </div>

      <div className="w-full rounded-lg border border-border bg-muted/40 p-4 text-sm text-muted-foreground">
        <p>如果没有收到邮件，请检查垃圾邮件文件夹，或点击下方按钮重新发送。</p>
      </div>

      {resendMsg && (
        <p className="text-sm text-muted-foreground">{resendMsg}</p>
      )}

      <div className="flex w-full flex-col gap-3">
        <Button
          variant="outline"
          className="h-10 w-full"
          onClick={resend}
          disabled={resending || !email}
        >
          {resending ? '发送中...' : '重新发送验证邮件'}
        </Button>

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
    </div>
  )
}
