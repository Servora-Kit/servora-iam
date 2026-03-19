import { useState, useCallback } from 'react'
import { iamClients } from '#/api'

export interface UseResendVerificationReturn {
  resend: () => Promise<void>
  resending: boolean
  message: string
}

/**
 * 重发验证邮件 hook。
 * 在 login / register-success / verify-email 三处统一调用，消除重复逻辑。
 *
 * @param email 目标邮箱；为空时 resend() 直接返回
 */
export function useResendVerification(
  email: string,
): UseResendVerificationReturn {
  const [resending, setResending] = useState(false)
  const [message, setMessage] = useState('')

  const resend = useCallback(async () => {
    if (!email) return
    setResending(true)
    setMessage('')
    try {
      await iamClients.authn.RequestEmailVerification({ email })
      setMessage('验证邮件已重新发送，请检查收件箱')
    } catch {
      setMessage('发送失败，请稍后重试')
    } finally {
      setResending(false)
    }
  }, [email])

  return { resend, resending, message }
}
