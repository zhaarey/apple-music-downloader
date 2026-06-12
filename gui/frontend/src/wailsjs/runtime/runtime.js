export function EventsOn(eventName, callback) {
  if (window.runtime?.EventsOn) {
    return window.runtime.EventsOn(eventName, callback)
  }
  return () => {}
}

export function EventsOff(eventName, ...args) {
  window.runtime?.EventsOff?.(eventName, ...args)
}

export function EventsEmit(eventName, ...args) {
  window.runtime?.EventsEmit?.(eventName, ...args)
}

export function OnFileDrop(callback, useDropTarget) {
  return window.runtime?.OnFileDrop?.(callback, useDropTarget)
}

export function OnFileDropOff() {
  return window.runtime?.OnFileDropOff?.()
}
