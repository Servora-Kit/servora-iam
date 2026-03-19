import '@cap.js/widget'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useReducer, useRef } from 'react'
import { Button } from '#/components/ui/button'
import { Input } from '#/components/ui/input'
import { Label } from '#/components/ui/label'
import { iamClients } from '#/api'
import type { ApiError } from '@servora/web-pkg/request'
import { isKratosReason, kratosMessage } from '@servora/web-pkg/errors'

export const Route = createFileRoute('/_auth/register')({
  component: RegisterPage,
})

// ---------- State / Reducer ----------

type Status = 'idle' | 'submitting'

interface RegisterState {
  name: string
  email: string
  password: string
  passwordConfirm: string
  status: Status
  error: string
  capResetKey: number
}

type RegisterAction =
  | {
      type: 'SET_FIELD'
      field: 'name' | 'email' | 'password' | 'passwordConfirm'
      value: string
    }
  | { type: 'SUBMIT' }
  | { type: 'SUBMIT_ERROR'; error: string }
  | { type: 'RESET_CAP' }
  | { type: 'CLEAR_ERROR' }

const initialState: RegisterState = {
  name: '',
  email: '',
  password: '',
  passwordConfirm: '',
  status: 'idle',
  error: '',
  capResetKey: 0,
}

function reducer(state: RegisterState, action: RegisterAction): RegisterState {
  switch (action.type) {
    case 'SET_FIELD':
      return { ...state, [action.field]: action.value }
    case 'SUBMIT':
      return { ...state, status: 'submitting', error: '' }
    case 'SUBMIT_ERROR':
      return { ...state, status: 'idle', error: action.error }
    case 'RESET_CAP':
      return {
        ...state,
        status: 'idle',
        error: '人机验证已过期，请重新验证',
        capResetKey: state.capResetKey + 1,
      }
    case 'CLEAR_ERROR':
      return { ...state, error: '' }
    default:
      return state
  }
}

// ---------- Component ----------

function RegisterPage() {
  const navigate = useNavigate()
  const [state, dispatch] = useReducer(reducer, initialState)
  const capTokenRef = useRef<string>('')
  const mounted = useRef(false)

  useEffect(() => {
    mounted.current = true
  }, [])

  // Cap widget 派发 "solve" 事件，detail 为 { token: string }
  const handleCapRef = useCallback((node: HTMLElement | null) => {
    if (!node) return
    node.addEventListener('solve', (e: Event) => {
      const token = (e as CustomEvent<{ token: string }>).detail.token
      if (token) {
        capTokenRef.current = token
        dispatch({ type: 'CLEAR_ERROR' })
      }
    })
  }, [])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    if (state.password !== state.passwordConfirm) {
      dispatch({ type: 'SUBMIT_ERROR', error: '两次输入的密码不一致' })
      return
    }
    if (!capTokenRef.current) {
      dispatch({ type: 'SUBMIT_ERROR', error: '请先完成人机验证' })
      return
    }

    dispatch({ type: 'SUBMIT' })
    try {
      await iamClients.authn.SignupByEmail({
        name: state.name,
        email: state.email,
        password: state.password,
        passwordConfirm: state.passwordConfirm,
        capToken: capTokenRef.current,
      })
      void navigate({ to: '/register-success', search: { email: state.email } })
    } catch (err: unknown) {
      const apiErr = err as ApiError
      if (isKratosReason(apiErr, 'INVALID_CAPTCHA')) {
        capTokenRef.current = ''
        dispatch({ type: 'RESET_CAP' })
      } else {
        dispatch({
          type: 'SUBMIT_ERROR',
          error: kratosMessage(apiErr, '注册失败，请稍后重试'),
        })
      }
    }
  }

  const loading = state.status === 'submitting'

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-3xl font-bold text-foreground">创建账号</h1>
        <p className="mt-2 text-muted-foreground">注册 Servora IAM 管理平台</p>
      </div>

      {state.error && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {state.error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="name">用户名</Label>
          <Input
            id="name"
            type="text"
            value={state.name}
            onChange={(e) =>
              dispatch({ type: 'SET_FIELD', field: 'name', value: e.target.value })
            }
            placeholder="至少 5 个字符"
            required
            minLength={5}
            autoFocus
            className="h-10"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="email">邮箱</Label>
          <Input
            id="email"
            type="email"
            value={state.email}
            onChange={(e) =>
              dispatch({
                type: 'SET_FIELD',
                field: 'email',
                value: e.target.value,
              })
            }
            placeholder="you@example.com"
            required
            className="h-10"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="password">密码</Label>
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
            className="h-10"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="passwordConfirm">确认密码</Label>
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
            placeholder="再次输入密码"
            required
            minLength={6}
            maxLength={20}
            className="h-10"
          />
        </div>

        {/* Cap PoW CAPTCHA — key 变化时强制重新挂载，无需 DOM 操作 */}
        <div className="flex justify-center">
          <cap-widget
            key={state.capResetKey}
            ref={handleCapRef}
            data-cap-api-endpoint="/v1/cap/"
          />
        </div>

        <Button type="submit" className="mt-2 h-10 w-full" disabled={loading}>
          {loading ? '注册中...' : '注册'}
        </Button>
      </form>

      <p className="text-center text-sm text-muted-foreground">
        已有账号？{' '}
        <Link
          to="/login"
          search={{ redirect: '', authRequestID: '' }}
          className="font-medium text-primary hover:underline"
        >
          立即登录
        </Link>
      </p>
    </div>
  )
}
