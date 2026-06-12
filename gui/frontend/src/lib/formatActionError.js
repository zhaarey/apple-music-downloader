export function formatActionError(error, action) {
  const msg = error?.message || String(error ?? 'Unknown error')
  return `${action} failed: ${msg}`
}
