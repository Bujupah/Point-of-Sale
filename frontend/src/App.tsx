import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { I18nProvider } from './i18n'
import { ErrorBoundary } from './app/ErrorBoundary'
import { AuthProvider, useAuth } from './app/AuthContext'
import { SettingsProvider } from './app/SettingsContext'
import { ShiftProvider } from './app/ShiftContext'
import { SellProvider } from './app/SellContext'
import { LoginScreen } from './features/auth/LoginScreen'
import { LockScreen } from './features/auth/LockScreen'
import { Shell } from './app/Shell'
import { SellScreen } from './features/sell/SellScreen'
import { OrdersScreen } from './features/orders/OrdersScreen'
import { CustomersScreen } from './features/customers/CustomersScreen'
import { RegisterScreen } from './features/register/RegisterScreen'
import { ProductsScreen } from './features/catalog/ProductsScreen'
import { ReportsScreen } from './features/reports/ReportsScreen'
import { SettingsScreen } from './features/settings/SettingsScreen'

function Gate() {
  const { loading, user, locked } = useAuth()

  if (loading) {
    return (
      <div className="auth-screen">
        <div className="spinner" />
      </div>
    )
  }
  if (!user) return <LoginScreen />

  // LockScreen and the main app both need settings/shift/cart context (it
  // shows the register name, and unlocking must not lose the parked cart),
  // so both branches render inside the same provider tree rather than
  // LockScreen being swapped in above it — rendering it outside these
  // providers is what caused the blank-page crash: it calls useShift(),
  // which throws with no ShiftProvider ancestor, and with no error boundary
  // that unmounts the whole app.
  return (
    <SettingsProvider>
      <ShiftProvider>
        <SellProvider>
          {locked ? (
            <LockScreen />
          ) : (
            <BrowserRouter>
              <Routes>
                <Route path="/" element={<Shell />}>
                  <Route index element={<Navigate to="/sell" replace />} />
                  <Route path="sell" element={<SellScreen />} />
                  <Route path="orders" element={<OrdersScreen />} />
                  <Route path="customers" element={<CustomersScreen />} />
                  <Route path="register" element={<RegisterScreen />} />
                  <Route path="products" element={<ProductsScreen />} />
                  <Route path="reports" element={<ReportsScreen />} />
                  <Route path="settings" element={<SettingsScreen />} />
                  <Route path="*" element={<Navigate to="/sell" replace />} />
                </Route>
              </Routes>
            </BrowserRouter>
          )}
        </SellProvider>
      </ShiftProvider>
    </SettingsProvider>
  )
}

export default function App() {
  return (
    <ErrorBoundary>
      <I18nProvider>
        <AuthProvider>
          <Gate />
        </AuthProvider>
      </I18nProvider>
    </ErrorBoundary>
  )
}
