import { useCallback, useEffect, useRef, useState } from 'react'
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import { TagResolveDrop } from '../../wailsjs/go/main/App'

export const TAG_EDITOR_DROP_CLASS = 'tag-editor-drop-target'

export function useTagEditorDrop({ onSingleFile, onAlbumFolder, onError, disabled = false, active = true }) {
  const [dragOver, setDragOver] = useState(false)
  const dragDepthRef = useRef(0)
  const disabledRef = useRef(disabled)
  const callbacksRef = useRef({ onSingleFile, onAlbumFolder, onError })

  disabledRef.current = disabled
  callbacksRef.current = { onSingleFile, onAlbumFolder, onError }

  const handleDrop = useCallback(async (paths) => {
    if (disabledRef.current || !paths?.length) return
    setDragOver(false)
    dragDepthRef.current = 0
    try {
      const plan = await TagResolveDrop(paths)
      if (plan.mode === 'album') {
        await callbacksRef.current.onAlbumFolder?.(plan.path, plan.message)
      } else {
        await callbacksRef.current.onSingleFile?.(plan.path, plan.message)
      }
    } catch (e) {
      callbacksRef.current.onError?.(String(e?.message || e))
    }
  }, [])

  useEffect(() => {
    if (!active) {
      OnFileDropOff()
      setDragOver(false)
      dragDepthRef.current = 0
      return undefined
    }
    OnFileDrop((_x, _y, paths) => {
      void handleDrop(paths)
    }, true)
    return () => OnFileDropOff()
  }, [active, handleDrop])

  const onDragEnter = useCallback((e) => {
    if (disabledRef.current) return
    e.preventDefault()
    e.stopPropagation()
    dragDepthRef.current += 1
    setDragOver(true)
  }, [])

  const onDragLeave = useCallback((e) => {
    if (disabledRef.current) return
    e.preventDefault()
    e.stopPropagation()
    const related = e.relatedTarget
    if (related && e.currentTarget.contains(related)) return
    dragDepthRef.current -= 1
    if (dragDepthRef.current <= 0) {
      dragDepthRef.current = 0
      setDragOver(false)
    }
  }, [])

  const onDragOver = useCallback((e) => {
    if (disabledRef.current) return
    e.preventDefault()
    e.stopPropagation()
  }, [])

  return {
    dragOver,
    dropHandlers: { onDragEnter, onDragLeave, onDragOver },
    dropTargetClassName: TAG_EDITOR_DROP_CLASS,
  }
}
