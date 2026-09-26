// Runtime capability flags for the two build tiers described in
// docs/01-architecture-and-repo-structure.md: the modern WebView2 tier
// (this build) and the legacy CEF-XP/IE8 tier. Components read these flags
// instead of sniffing the user agent, so the legacy tier (built separately,
// see the architecture doc's build model) can flip them off without any
// component-level changes.
export const capabilities = {
  animations: true,
  cssGrid: true,
  asyncNative: true,
  webAudio: typeof window !== 'undefined' && 'AudioContext' in window,
}

export type Capabilities = typeof capabilities
