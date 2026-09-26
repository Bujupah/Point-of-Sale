import { useCallback, useEffect, useRef, useState } from 'react'
import { useI18n } from '../../i18n'
import { useShift } from '../../app/ShiftContext'
import { useSell } from '../../app/SellContext'
import { api, ApiError } from '../../services/api'
import { audioService } from '../../services/audio'
import { CategoryBar } from './CategoryBar'
import { ProductGrid } from './ProductGrid'
import { VariantModal } from './VariantModal'
import { CartPanel } from '../cart/CartPanel'
import { CheckoutModal } from '../checkout/CheckoutModal'
import { SaleCompleteScreen } from '../checkout/SaleCompleteScreen'
import { OpenShiftPrompt } from '../register/OpenShiftPrompt'
import { useBarcodeScanner, type ScannerStatus } from '../../hooks/useBarcodeScanner'
import { HeldSalesModal } from '../cart/HeldSalesModal'
import type { CartModifierSelection, CartVariantSelection, Product, Sale } from '../../types'
import { computeCartTotals } from '../../utils/pricing'

const statusLabels: Record<ScannerStatus, string> = {
  READY: 'scanner_ready',
  SCANNING: 'scanner_scanning',
  FOUND: 'scanner_found',
  NOT_FOUND: 'scanner_not_found',
  DISABLED: 'scanner_disabled',
}

export function SellScreen() {
  const { t } = useI18n()
  const { shift, loading: shiftLoading } = useShift()
  const { addProduct, lines, orderDiscountType, orderDiscountAmount, orderDiscountPercentBps, holdCart, orderNote } = useSell()

  const [query, setQuery] = useState('')
  const [categoryId, setCategoryId] = useState<number | 'all' | 'favorites'>('all')
  const [reloadToken, setReloadToken] = useState(0)
  const [variantProduct, setVariantProduct] = useState<Product | null>(null)
  const [checkoutTotals, setCheckoutTotals] = useState<ReturnType<typeof computeCartTotals> | null>(null)
  const [completedSale, setCompletedSale] = useState<Sale | null>(null)
  const [printError, setPrintError] = useState<string | undefined>(undefined)
  const [heldModalOpen, setHeldModalOpen] = useState(false)
  const searchRef = useRef<HTMLInputElement>(null)

  // Keyboard-first workflow (brief §84): the sell screen owns the shortcuts
  // that act on the whole sale (search focus, hold, held list, payment,
  // drawer). Discount/customer shortcuts live inside CartPanel's own menu
  // for now since they need its local modal state.
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'F2') {
        e.preventDefault()
        searchRef.current?.focus()
      } else if (e.key === 'F5') {
        e.preventDefault()
        if (lines.length > 0) holdCart(orderNote, '')
      } else if (e.key === 'F6') {
        e.preventDefault()
        setHeldModalOpen(true)
      } else if (e.key === 'F8') {
        e.preventDefault()
        if (lines.length > 0) {
          setCheckoutTotals(computeCartTotals(lines, orderDiscountType, orderDiscountAmount, orderDiscountPercentBps))
        }
      } else if (e.key === 'F11') {
        e.preventDefault()
        api.post('/api/hardware/drawer/open').catch(() => {})
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [lines, orderDiscountType, orderDiscountAmount, orderDiscountPercentBps, holdCart, orderNote])

  // The product grid/search list intentionally omits variant and modifier
  // group data (they'd mean an extra 2-3 queries per row for a list that
  // can have hundreds of products — see catalog.Service.SearchProducts).
  // So a click re-fetches the single, fully-hydrated product before
  // deciding whether it needs the variant modal or can go straight into
  // the cart; this is one fast local round-trip, not a per-row cost.
  const handleSelect = useCallback(
    async (product: Product) => {
      let full = product
      try {
        full = await api.get<Product>(`/api/products/${product.id}`)
      } catch {
        // Fall back to the list version; worst case a variant/modifier
        // product briefly adds without its modal, which the cashier can
        // still fix via the cart line editor.
      }
      if ((full.variant_groups?.length ?? 0) > 0 || (full.modifier_groups?.length ?? 0) > 0) {
        setVariantProduct(full)
      } else {
        addProduct(full, [], [])
        audioService.click()
      }
    },
    [addProduct],
  )

  const handleScan = useCallback(
    async (barcode: string): Promise<boolean> => {
      try {
        const product = await api.get<Product>(`/api/products/barcode/${encodeURIComponent(barcode)}`)
        handleSelect(product)
        audioService.scanSuccess()
        return true
      } catch (err) {
        if (err instanceof ApiError) audioService.scanError()
        return false
      }
    },
    [handleSelect],
  )

  const scannerStatus = useBarcodeScanner({ enabled: !!shift, onScan: handleScan })

  function confirmVariants(variants: CartVariantSelection[], modifiers: CartModifierSelection[], quantity: number, notes: string) {
    if (variantProduct) addProduct(variantProduct, variants, modifiers, notes || undefined, quantity)
    setVariantProduct(null)
    audioService.click()
  }

  if (shiftLoading) {
    return (
      <div className="empty-state" style={{ width: '100%' }}>
        <div className="spinner" />
      </div>
    )
  }

  if (!shift) {
    return <OpenShiftPrompt />
  }

  return (
    <div className="sell-screen">
      <div className="sell-left">
        <div className="sell-search-row">
          <input
            ref={searchRef}
            className="input sell-search"
            placeholder={t('search_placeholder')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            data-scanner-passthrough="true"
          />
          <div className={`scanner-indicator scanner-${scannerStatus.toLowerCase()}`}>{t(statusLabels[scannerStatus] as any)}</div>
        </div>
        <CategoryBar activeId={categoryId} onChange={setCategoryId} />
        <div className="product-grid-scroll">
          <ProductGrid query={query} categoryId={categoryId} onSelect={handleSelect} reloadToken={reloadToken} />
        </div>
      </div>
      <CartPanel onCharge={(totals) => setCheckoutTotals(totals)} />

      {variantProduct && <VariantModal product={variantProduct} onConfirm={confirmVariants} onClose={() => setVariantProduct(null)} />}

      {checkoutTotals && lines.length > 0 && (
        <CheckoutModal
          totals={checkoutTotals}
          onClose={() => setCheckoutTotals(null)}
          onComplete={(sale, printErr) => {
            setCheckoutTotals(null)
            setCompletedSale(sale)
            setPrintError(printErr)
            audioService.saleComplete()
            setReloadToken((v) => v + 1)
          }}
        />
      )}

      {heldModalOpen && <HeldSalesModal onClose={() => setHeldModalOpen(false)} />}

      {completedSale && (
        <SaleCompleteScreen
          sale={completedSale}
          printError={printError}
          onNewSale={() => {
            setCompletedSale(null)
            setPrintError(undefined)
          }}
        />
      )}
    </div>
  )
}
