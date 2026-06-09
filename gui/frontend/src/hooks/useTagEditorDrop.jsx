import { useCallback, useEffect, useRef, useState } from 'react'
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import { TagResolveDrop } from '../../wailsjs/go/main/App'

export const TAG_EDITOR_DROP_CLASS = 'tag-editor-drop-target'

export function useTagEditorDrop({ onSingleFile, onAlbumFolder, onError, disabled = false }) {
  const [dragOver, setDragOver] = useState(false)
  const dragDepthRef = useRef(0)

  const handleDrop = useCallback(
    async (paths) => {
      if (disabled || !paths?.length) return
      setDragOver(false)
      dragDepthRef.current = 0
      try {
        const plan = await TagResolveDrop(paths)
        if (plan.mode === 'album') {
          await onAlbumFolder?.(plan.path, plan.message)
        } else {
          await onSingleFile?.(plan.path, plan.message)
        }
      } catch (e) {
        onError?.(String(e?.message || e))
      }
    },
    [disabled, onSingleFile, onAlbumFolder, onError],
  )

  useEffect(() => {
    if (disabled) {
      OnFileDropOff()
      return undefined
    }
    OnFileDrop((_x, _y, paths) => {
      handleDrop(paths)
    }, true)
    return () => OnFileDropOff()
  }, [disabled, handleDrop])

  const onDragEnter = useCallback(
    (e) => {
      if (disabled) return
      e.preventDefault()
      e.stopPropagation()
      dragDepthRef.current += 1
      setDragOver(true)
    },
    [disabled],
  )

  const onDragLeave = useCallback(
    (e) => {
      if (disabled) return
      e.preventDefault()
      e.stopPropagation()
      dragDepthRef.current -= 1
      if (dragDepthRef.current <= 0) {
        dragDepthRef.current = 0
        setDragOver(false)
      }
    },
    [disabled],
  )

  const onDragOver = useCallback(
    (e) => {
      if (disabled) return
      e.preventDefault()
      e.stopPropagation()
    },
    [disabled],
  )

  const dropZoneProps = {
    className: TAG_EDITOR_DROP_CLASS,
    onDragEnter,
    onDragLeave,
    onDragOver,
  }

  return { dragOver, dropZoneProps }
}
