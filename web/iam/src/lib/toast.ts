/**
 * 统一消息提示工具，基于 sonner 封装。
 *
 * 设计原则：
 * - toast.promise()  → 自动处理 loading / success / error，并标记 error 已处理，防止全局 onError 重复弹出
 * - toast.fromApiError() → 全局 onError 钩子调用，自动解析后端 reason/message 字段
 * - 401 / 已处理的 error 静默忽略
 */

import { toast as sonner } from 'sonner'
import type { ExternalToast } from 'sonner'
import type { ApiError } from '@servora/web-pkg/request'
import { kratosMessage } from '@servora/web-pkg/errors'

// ---------- 防重复机制 ----------
// 用 WeakSet 标记已被 toast.promise 处理过的 error，避免全局 onError 再次弹出
const _handled = new WeakSet<Error>()

function _mark(err: unknown) {
  if (err instanceof Error) _handled.add(err)
}

export function isHandledError(err: unknown): boolean {
  return err instanceof Error && _handled.has(err)
}

// ---------- 对外 API ----------
const toast = {
  success(message: string, opts?: ExternalToast) {
    sonner.success(message, opts)
  },

  error(message: string, opts?: ExternalToast) {
    sonner.error(message, opts)
  },

  warning(message: string, opts?: ExternalToast) {
    sonner.warning(message, opts)
  },

  info(message: string, opts?: ExternalToast) {
    sonner.info(message, opts)
  },

  /**
   * 包装 Promise，自动显示 loading → success/error toast。
   * error 会被标记为已处理，全局 onError 不会重复弹出。
   */
  promise<T>(
    promise: Promise<T>,
    opts: {
      loading: string
      success: string | ((data: T) => string)
      error?: string | ((err: unknown) => string)
    },
  ): Promise<T> {
    // 标记 error，防止全局 onError 重复 toast
    const wrapped = promise.catch((err: unknown) => {
      _mark(err)
      throw err
    })

    void sonner.promise(wrapped, {
      loading: opts.loading,
      success: opts.success as string,
      error: opts.error
        ? typeof opts.error === 'string'
          ? opts.error
          : (err: unknown) => (opts.error as (e: unknown) => string)(err)
        : (err: unknown) =>
            kratosMessage(err as ApiError, '操作失败'),
    })

    // 返回原始 promise（让调用方可以 await）
    return promise
  },

  /**
   * 全局 onError 钩子调用。
   * - 401 由登录过期 Dialog 处理，静默忽略
   * - 已被 toast.promise 标记的 error 静默忽略
   */
  fromApiError(err: ApiError) {
    if (err.httpStatus === 401) return
    if (isHandledError(err)) return
    sonner.error(kratosMessage(err))
  },
}

export { toast }
