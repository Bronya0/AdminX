import { onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Modal, message } from 'ant-design-vue'
import { useUserStore } from '@/stores/user'
import { refreshAccessToken } from '@/utils/request'

export interface BusinessPeer { win: Window; origin: string }
interface BridgeHooks {
  ready?: () => void
  route?: (hash: string) => void
  dirty?: (value: boolean) => void
  initialRoute?: () => string
}

// 平台只暴露固定能力；不代理任意 URL 请求，不接收 HTML、脚本或组件。
export function useBusinessBridge(peers: () => BusinessPeer[], hooks: BridgeHooks = {}) {
  const store = useUserStore()
  const router = useRouter()
  let disposed = false
  let lastActivity = 0
  let closeConfirm: (() => void) | undefined
  const pending = new Set<string>()
  const send = (peer: BusinessPeer, data: unknown) => peer.win.postMessage(data, peer.origin)
  const broadcast = (event: string, payload?: unknown) => {
    peers().forEach(peer => send(peer, { type: 'adminx:event', event, payload }))
  }
  const menus = () => store.menus.filter(m =>
    m.menu_type !== 'button' && m.path &&
    (m.menu_type === 'iframe' || (m.path.startsWith('/') && !m.path.startsWith('//') && !m.path.includes(String.fromCharCode(92)))),
  )
  async function accessToken(rejectedToken?: unknown) {
    if (!store.token) throw new Error('请先登录平台')
    let exp = 0
    try { exp = JSON.parse(atob(store.token.split('.')[1]!.replace(/-/g, '+').replace(/_/g, '/'))).exp } catch { /* 交给刷新校验 */ }
    if ((typeof rejectedToken === 'string' && rejectedToken === store.token) || !exp || exp * 1000 < Date.now() + 30000) {
      return refreshAccessToken()
    }
    return store.token
  }
  async function dispatch(method: string, payload: Record<string, unknown>) {
    switch (method) {
      case 'context':
        return {
          token: await accessToken(),
          theme: JSON.parse(JSON.stringify(store.theme)),
          user: store.user ? { id: store.user.id, username: store.user.username } : null,
          menus: menus().map(m => ({ code: m.code, name: m.name })),
          route: hooks.initialRoute?.() || '',
        }
      case 'token': return { token: await accessToken(payload.rejectedToken) }
      case 'ready': hooks.ready?.(); return null
      case 'activity':
        if (Date.now() - lastActivity > 3000) {
          lastActivity = Date.now()
          window.dispatchEvent(new Event('adminx:business-activity'))
        }
        return null
      case 'dirty': hooks.dirty?.(payload.value === true); return null
      case 'route': {
        const hash = payload.hash
        if (typeof hash !== 'string' || hash.length > 2048 || (hash && !hash.startsWith('#')) || /[\r\n]/.test(hash)) throw new Error('无效的业务路由')
        hooks.route?.(hash)
        return null
      }
      case 'navigate': {
        const target = menus().find(m => m.code === payload.code)
        if (!target) throw new Error('目标菜单不存在或无权访问')
        if (target.menu_type === 'iframe') {
          await router.push({ path: '/iframe', query: { url: target.path, title: target.name } })
        } else {
          await router.push(target.path)
        }
        return null
      }
      case 'confirm':
        if (closeConfirm) throw new Error('请先处理当前确认框')
        return new Promise<boolean>(resolve => {
          const finish = (value: boolean) => { closeConfirm = undefined; resolve(value) }
          const modal = Modal.confirm({
            title: '操作确认',
            okText: '确定', cancelText: '取消',
            content: typeof payload.text === 'string' ? payload.text.slice(0, 500) : '是否继续？',
            onOk: () => finish(true), onCancel: () => finish(false),
          })
          closeConfirm = () => { modal.destroy(); finish(false) }
        })
      case 'message': {
        const text = typeof payload.text === 'string' ? payload.text.slice(0, 200) : ''
        if (text) message.info(text)
        return null
      }
      default: throw new Error('平台不支持该能力')
    }
  }
  const onMessage = async (event: MessageEvent) => {
    const peer = peers().find(p => p.win === event.source && p.origin === event.origin)
    if (!peer || disposed) return
    const data = event.data
    // 保留原有接入协议，旧业务页仍可获取 token。
    if (data?.type === 'adminx:request-token') {
      try { send(peer, { type: 'adminx:token', token: await accessToken() }); hooks.ready?.() } catch { /* 新协议有明确错误响应 */ }
      return
    }
    if (data?.type !== 'adminx:request' || data.version !== 1 || typeof data.id !== 'string' || data.id.length > 100 || typeof data.method !== 'string') return
    if (pending.has(data.id) || pending.size >= 20) return
    pending.add(data.id)
    try {
      if (!store.token) throw new Error('会话已结束')
      const result = await dispatch(data.method, data.payload && typeof data.payload === 'object' ? data.payload : {})
      if (!disposed && peers().some(p => p.win === peer.win && p.origin === peer.origin)) {
        send(peer, { type: 'adminx:response', id: data.id, result })
      }
    } catch (error) {
      if (!disposed) send(peer, { type: 'adminx:response', id: data.id, error: error instanceof Error ? error.message : '平台请求失败' })
    } finally { pending.delete(data.id) }
  }
  window.addEventListener('message', onMessage)
  watch(() => store.theme, value => broadcast('theme', JSON.parse(JSON.stringify(value))), { deep: true })
  watch(() => store.token, (value, old) => {
    if (!value && old) { closeConfirm?.(); hooks.dirty?.(false); broadcast('logout') }
  }, { flush: 'sync' })
  onUnmounted(() => {
    broadcast('disconnect')
    disposed = true
    closeConfirm?.()
    window.removeEventListener('message', onMessage)
  })
}
