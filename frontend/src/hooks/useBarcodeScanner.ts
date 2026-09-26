import { useEffect, useRef, useState } from 'react'
import { useSettings } from '../app/SettingsContext'

export type ScannerStatus = 'READY' | 'SCANNING' | 'FOUND' | 'NOT_FOUND' | 'DISABLED'

interface Options {
  enabled: boolean
  onScan: (barcode: string) => Promise<boolean> // resolves true if a product was found
}

// Detects a USB barcode scanner acting as a keyboard: characters arrive far
// faster than a human can type, terminated by a suffix key (brief §12).
// Works globally while the Sell screen is mounted — no need to focus a
// search box first.
export function useBarcodeScanner({ enabled, onScan }: Options) {
  const { settings } = useSettings()
  const [status, setStatus] = useState<ScannerStatus>('READY')
  const bufferRef = useRef('')
  const lastKeyTimeRef = useRef(0)
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const suffixKey = settings['scanner.suffix_key'] || 'Enter'
  const maxIntervalMs = Number(settings['scanner.max_interval_ms'] || '50')
  const minLength = Number(settings['scanner.min_length'] || '6')

  useEffect(() => {
    if (!enabled) {
      setStatus('DISABLED')
      return
    }
    setStatus('READY')

    function onKeyDown(e: KeyboardEvent) {
      const target = e.target as HTMLElement | null
      const typingInField = target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) && target.getAttribute('data-scanner-passthrough') !== 'true'

      const now = performance.now()
      const elapsed = now - lastKeyTimeRef.current
      lastKeyTimeRef.current = now

      if (e.key === suffixKey) {
        const code = bufferRef.current
        bufferRef.current = ''
        if (code.length >= minLength) {
          if (typingInField) return // let a manual Enter in a real form field behave normally
          e.preventDefault()
          setStatus('SCANNING')
          onScan(code).then((found) => {
            setStatus(found ? 'FOUND' : 'NOT_FOUND')
            if (resetTimerRef.current) clearTimeout(resetTimerRef.current)
            resetTimerRef.current = setTimeout(() => setStatus('READY'), 1200)
          })
        }
        return
      }

      if (e.key.length !== 1) return // ignore modifier/navigation keys

      if (elapsed > maxIntervalMs) {
        bufferRef.current = '' // gap too large: this is human typing, not a scan
      }
      if (!typingInField) {
        bufferRef.current += e.key
        if (bufferRef.current.length === 1) setStatus('SCANNING')
      }
    }

    window.addEventListener('keydown', onKeyDown, true)
    return () => {
      window.removeEventListener('keydown', onKeyDown, true)
      if (resetTimerRef.current) clearTimeout(resetTimerRef.current)
    }
  }, [enabled, onScan, suffixKey, maxIntervalMs, minLength])

  return status
}
