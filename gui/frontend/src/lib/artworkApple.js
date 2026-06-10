export function optimizedPreviewFromAnalysis(analysis) {
  if (!analysis?.optimized_b64 || !analysis?.optimized_mime) return ''
  return `data:${analysis.optimized_mime};base64,${analysis.optimized_b64}`
}

export async function loadArtworkAnalysis(path, analyzeFn) {
  if (!path) return null
  try {
    return await analyzeFn(path)
  } catch {
    return null
  }
}

export async function loadEmbeddedArtworkAnalysis(audioPath, analyzeEmbeddedFn) {
  if (!audioPath) return null
  try {
    return await analyzeEmbeddedFn(audioPath)
  } catch {
    return null
  }
}
