import { useState } from 'react'
import { Check } from 'lucide-react'
import { useI18n, type Language } from '../../i18n'
import { useAuth } from '../../app/AuthContext'
import { useSettings } from '../../app/SettingsContext'

export function StoreSettingsTab() {
  const { t, language, setLanguage } = useI18n()
  const { hasPermission } = useAuth()
  const { settings, save } = useSettings()
  const canWrite = hasPermission('settings.write')

  const [storeName, setStoreName] = useState(settings['store.name'] ?? '')
  const [address, setAddress] = useState(settings['store.address'] ?? '')
  const [taxId, setTaxId] = useState(settings['store.tax_id'] ?? '')
  const [currencySymbol, setCurrencySymbol] = useState(settings['currency.symbol'] ?? '€')
  const [currencyDecimals, setCurrencyDecimals] = useState(settings['currency.decimals'] ?? '2')
  const [loyaltyRate, setLoyaltyRate] = useState(settings['loyalty.rate_per_100'] ?? '1')
  const [discountLimit, setDiscountLimit] = useState(String(Number(settings['discount.max_percent_bps'] ?? '2000') / 100))
  const [saved, setSaved] = useState(false)

  async function submit() {
    await save({
      'store.name': storeName,
      'store.address': address,
      'store.tax_id': taxId,
      'currency.symbol': currencySymbol,
      'currency.decimals': currencyDecimals,
      'loyalty.rate_per_100': loyaltyRate,
      'discount.max_percent_bps': String(Math.round(Number(discountLimit) * 100)),
    })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div className="card" style={{ padding: 24, maxWidth: 480 }}>
      <div className="stack">
        <label className="field">{t('settings_language')}
          <select className="input" value={language} onChange={(e) => setLanguage(e.target.value as Language)}>
            <option value="es">Español</option>
            <option value="en">English</option>
            <option value="ar">العربية</option>
          </select>
        </label>
        <label className="field">Store name<input className="input" disabled={!canWrite} value={storeName} onChange={(e) => setStoreName(e.target.value)} /></label>
        <label className="field">Address<input className="input" disabled={!canWrite} value={address} onChange={(e) => setAddress(e.target.value)} /></label>
        <label className="field">Tax ID<input className="input" disabled={!canWrite} value={taxId} onChange={(e) => setTaxId(e.target.value)} /></label>
        <div className="form-grid-2">
          <label className="field">Currency symbol<input className="input" disabled={!canWrite} value={currencySymbol} onChange={(e) => setCurrencySymbol(e.target.value)} /></label>
          <label className="field">Decimals<select className="input" disabled={!canWrite} value={currencyDecimals} onChange={(e) => setCurrencyDecimals(e.target.value)}>
            <option value="2">2</option>
            <option value="3">3</option>
          </select></label>
        </div>
        <label className="field">Loyalty points per 100 spent<input className="input" disabled={!canWrite} value={loyaltyRate} onChange={(e) => setLoyaltyRate(e.target.value)} /></label>
        <label className="field">Cashier discount limit (%)<input className="input" disabled={!canWrite} value={discountLimit} onChange={(e) => setDiscountLimit(e.target.value)} /></label>
        {canWrite && (
          <button className="btn btn-primary" onClick={submit}>{saved ? <><Check size={16} /> Saved</> : t('common_save')}</button>
        )}
      </div>
    </div>
  )
}
