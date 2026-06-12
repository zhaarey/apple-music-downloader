import { useCallback, useState } from 'react'
import ConfirmDialog from '../components/ConfirmDialog'

export function useConfirm() {
  const [pending, setPending] = useState(null)

  const requestConfirm = useCallback(
    (options) =>
      new Promise((resolve) => {
        setPending({ ...options, resolve })
      }),
    [],
  )

  const close = useCallback((result) => {
    setPending((current) => {
      current?.resolve?.(result)
      return null
    })
  }, [])

  const ConfirmDialogSlot = pending ? (
    <ConfirmDialog
      open
      title={pending.title}
      message={pending.message}
      details={pending.details}
      confirmLabel={pending.confirmLabel}
      cancelLabel={pending.cancelLabel}
      destructive={pending.destructive !== false}
      busy={pending.busy}
      onConfirm={() => close(true)}
      onCancel={() => close(false)}
    />
  ) : null

  return { requestConfirm, ConfirmDialogSlot, setConfirmBusy: (busy) => setPending((p) => (p ? { ...p, busy } : p)) }
}
