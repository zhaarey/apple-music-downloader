const WINDOWS = {
  workflowTabs: [
    { id: 'apple', label: 'Apple Music' },
    { id: 'youtube', label: 'YouTube' },
    { id: 'splice', label: 'Split mix' },
    { id: 'trim', label: 'Trim' },
    { id: 'metadata', label: 'Tag Editor' },
  ],
  utilityTabs: [
    { id: 'activity', label: 'Activity' },
    { id: 'requirements', label: 'Requirements' },
    { id: 'settings', label: 'Settings' },
  ],
  multitaskTabs: new Set(['activity', 'splice', 'trim', 'metadata', 'requirements', 'settings']),
  showWizard: true,
  showSetupChecklist: false,
  showActivityLogInSettings: false,
  showAppleSearch: true,
  showSplitHandoff: true,
  showLosslessQualities: false,
  tagline: 'Download · split · trim · tag for Apple Music',
}

const DARWIN = {
  workflowTabs: [
    { id: 'apple', label: 'Apple Music' },
    { id: 'youtube', label: 'YouTube' },
    { id: 'trim', label: 'Trim' },
    { id: 'metadata', label: 'Tag Editor' },
  ],
  utilityTabs: [{ id: 'settings', label: 'Settings' }],
  multitaskTabs: new Set(['trim', 'metadata', 'settings']),
  showWizard: false,
  showSetupChecklist: true,
  showActivityLogInSettings: true,
  showAppleSearch: false,
  showSplitHandoff: false,
  showLosslessQualities: false,
  tagline: 'Download · trim · tag for Apple Music',
}

export function featuresForPlatform(goos) {
  if (goos === 'darwin') return DARWIN
  return WINDOWS
}

export function isTabEnabled(features, tabId) {
  return (
    features.workflowTabs.some((t) => t.id === tabId) || features.utilityTabs.some((t) => t.id === tabId)
  )
}
