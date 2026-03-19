import { createFileRoute, Link, useSearch } from '@tanstack/react-router'
import { useReducer } from 'react'
import { Button } from '#/components/ui/button'
import { Input } from '#/components/ui/input'
import { Label } from '#/components/ui/label'
import { iamClients } from '#/api'
import type { ApiError } from '@servora/web-pkg/request'
import { kratosMessage } from '@servora/web-pkg/errors'

export const Route = createFileRoute('/_auth/reset-password')({
  validateSearch: (search: Record<string, unknown>) => ({
    token: (search.token as string) || '',
  }),
  component: ResetPasswordPage,
})

// ---------- State / Reducer ----------

type Status = 'idle' | 'submitting' | 'done'

interface ResetState {
  email: string
  password: string
  passwordConfirm: string
  status: Status
  error: string
  successMsg: string
}

type ResetAction =
  | { type: 'SET_EMAIL'; value: string }
  | { type: 'SET_FIELD'; field: 'password' | 'passwordConfirm'; value: string }
  | { type: 'SUBMIT' }
  | { type: 'SUBMIT_ERROR'; error: string }
  | { type: 'DONE'; successMsg: string }

const initialState: ResetState = {
  email: '',
  password: '',
  passwordConfirm: '',
  status: 'idle',
  error: '',
  successMsg: '',
}

function reducer(state: ResetState, action: ResetAction): ResetState {
  switch (action.type) {
    case 'SET_EMAIL':
      return { ...state, email: action.value }
    case 'SET_FIELD':
      return { ...state, [action.field]: action.value }
    case 'SUBMIT':
      return { ...state, status: 'submitting', error: '' }
    case 'SUBMIT_ERROR':
      return { ...state, status: 'idle', error: action.error }
    case 'DONE':
      return { ...state, status: 'done', successMsg: action.successMsg }
    default:
      return state
  }
}

// ---------- Component ----------

function ResetPasswordPage() {
  const { token } = useSearch({ from: '/_auth/reset-password' })
  const [state, dispatch] = useReducer(reducer, initialState)
  const loading = state.status === 'submitting'

  // 请求重置邮件（无 token 时）
  async function handleRequestReset(e: React.FormEvent) {
    e.preventDefault()
    dispatch({ type: 'SUBMIT' })
    try {
      await iamClients.authn.RequestPasswordReset({ email: state.email })
      dispatch({
        type: 'DONE',
        successMsg: '如果该邮箱已注册，您将在几分钟内收到重置链接。请同时检查垃圾邮件文件夹。',
      })
    } catch (err: unknown) {
      dispatch({
        type: 'SUBMIT_ERROR',
        error: kratosMessage(err as ApiError, '发送失败，请稍后重试'),
      })
    }
  }

  // 提交新密码（有 token 时）
  async function handleResetPassword(e: React.FormEvent) {
    e.preventDefault()
    if (state.password !== state.passwordConfirm) {
      dispatch({ type: 'SUBMIT_ERROR', error: '两次输入的密码不一致' })
      return
    }
    dispatch({ type: 'SUBMIT' })
    try {
      await iamClients.authn.ResetPassword({
        token,
        newPassword: state.password,
        newPasswordConfirm: state.passwordConfirm,
      })
      dispatch({ type: 'DONE', successMsg: '密码已重置，请使用新密码登录' })
    } catch (err: unknown) {
      dispatch({
        type: 'SUBMIT_ERROR',
        error: kratosMessage(err as ApiError, '重置失败，请稍后重试'),
      })
    }
  }

  // 完成状态
  if (state.status === 'done') {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-3xl font-bold text-foreground">
            {token ? '密码已重置' : '请查收邮件'}
          </h1>
          <p className="mt-2 text-muted-foreground">{state.successMsg}</p>
        </div>
        <Link to="/login" search={{ redirect: '', authRequestID: '' }}>
          <Button className="h-10 w-full">去登录</Button>
        </Link>
      </div>
    )
  }

  // 有 token — 新密码表单
  if (token) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-3xl font-bold text-foreground">重置密码</h1>
          <p className="mt-2 text-muted-foreground">请输入您的新密码</p>
        </div>

        {state.error && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {state.error}
          </div>
        )}

        <form onSubmit={handleResetPassword} className="flex flex-col gap-4">
          <div className="space-y-2">
            <Label htmlFor="password">新密码</Label>
            <Input
              id="password"
              type="password"
              value={state.password}
              onChange={(e) =>
                dispatch({
                  type: 'SET_FIELD',
                  field: 'password',
                  value: e.target.value,
                })
              }
              placeholder="6-20 个字符"
              required
              minLength={6}
              maxLength={20}
              autoFocus
              className="h-10"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="passwordConfirm">确认新密码</Label>
            <Input
              id="passwordConfirm"
              type="password"
              value={state.passwordConfirm}
              onChange={(e) =>
                dispatch({
                  type: 'SET_FIELD',
                  field: 'passwordConfirm',
                  value: e.target.value,
                })
              }
              placeholder="再次输入新密码"
              required
              minLength={6}
              maxLength={20}
              className="h-10"
            />
          </div>

          <Button
            type="submit"
            className="mt-2 h-10 w-full"
            disabled={loading}
          >
            {loading ? '提交中...' : '确认重置'}
          </Button>
        </form>
      </div>
    )
  }

  // 无 token — 请求重置邮件表单
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-3xl font-bold text-foreground">忘记密码</h1>
        <p className="mt-2 text-muted-foreground">
          输入您的账号邮箱，我们将发送重置链接
        </p>
      </div>

      {state.error && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {state.error}
        </div>
      )}

      <form onSubmit={handleRequestReset} className="flex flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="email">邮箱</Label>
          <Input
            id="email"
            type="email"
            value={state.email}
            onChange={(e) =>
              dispatch({ type: 'SET_EMAIL', value: e.target.value })
            }
            placeholder="you@example.com"
            required
            autoFocus
            className="h-10"
          />
        </div>

        <Button
          type="submit"
          className="mt-2 h-10 w-full"
          disabled={loading}
        >
          {loading ? '发送中...' : '发送重置邮件'}
        </Button>
      </form>

      <p className="text-center text-sm text-muted-foreground">
        想起密码了？{' '}
        <Link
          to="/login"
          search={{ redirect: '', authRequestID: '' }}
          className="font-medium text-primary hover:underline"
        >
          返回登录
        </Link>
      </p>
    </div>
  )
}
