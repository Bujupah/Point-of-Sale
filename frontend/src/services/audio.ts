// AudioService abstraction (brief §58): Web Audio when available, silent
// no-op otherwise. The POS must function perfectly with audio unavailable,
// so every method is safe to call unconditionally.
import { capabilities } from '../capabilities'

interface AudioService {
  click(): void
  scanSuccess(): void
  scanError(): void
  saleComplete(): void
  warning(): void
}

let enabled = true
export function setSoundEnabled(v: boolean) {
  enabled = v
}

function tone(ctx: AudioContext, freq: number, durationMs: number, delayMs = 0) {
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()
  osc.frequency.value = freq
  osc.type = 'sine'
  gain.gain.setValueAtTime(0.15, ctx.currentTime + delayMs / 1000)
  gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + delayMs / 1000 + durationMs / 1000)
  osc.connect(gain)
  gain.connect(ctx.destination)
  osc.start(ctx.currentTime + delayMs / 1000)
  osc.stop(ctx.currentTime + delayMs / 1000 + durationMs / 1000)
}

let sharedContext: AudioContext | null = null
function getContext(): AudioContext | null {
  if (!capabilities.webAudio) return null
  if (!sharedContext) {
    try {
      sharedContext = new AudioContext()
    } catch {
      return null
    }
  }
  return sharedContext
}

function play(fn: (ctx: AudioContext) => void) {
  if (!enabled) return
  const ctx = getContext()
  if (!ctx) return
  try {
    fn(ctx)
  } catch {
    // Audio is a nicety; any failure here is silently ignored.
  }
}

export const audioService: AudioService = {
  click: () => play((ctx) => tone(ctx, 700, 30)),
  scanSuccess: () => play((ctx) => tone(ctx, 1200, 80)),
  scanError: () => play((ctx) => tone(ctx, 200, 200)),
  saleComplete: () =>
    play((ctx) => {
      tone(ctx, 900, 100, 0)
      tone(ctx, 1300, 150, 100)
    }),
  warning: () => play((ctx) => tone(ctx, 350, 250)),
}
