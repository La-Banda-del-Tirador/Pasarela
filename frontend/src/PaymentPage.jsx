import { useState, useEffect, useRef } from 'react'
import './PaymentPage.css'

// ── Helpers ────────────────────────────────────────────────────────────────────
const CURRENCY_SYMBOLS = { BOB: 'Bs.', USD: 'US$' }
const BANK_NAMES = {
  1: 'BNB', 2: 'Banco Mercantil Santa Cruz', 3: 'BCP',
  4: 'BISA', 5: 'Banco Económico', 6: 'Banco Ganadero',
  7: 'BancoSol', 8: 'Banco Fassil', 9: 'Banco Fortaleza', 10: 'Banco Prodem',
}

function fmt(amount, currency) {
  const sym = CURRENCY_SYMBOLS[currency] ?? currency
  return `${sym} ${amount.toFixed(2)}`
}

function fmtDate(s) {
  if (!s) return ''
  return new Date(s + 'T12:00:00').toLocaleDateString('es-BO', {
    day: 'numeric', month: 'long', year: 'numeric',
  })
}

function fmtDateTime(s) {
  if (!s) return ''
  return new Date(s).toLocaleString('es-BO', {
    day: 'numeric', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

// ── Animated checkmark SVG ────────────────────────────────────────────────────
function CheckMark() {
  return (
    <div className="check-wrap" aria-hidden="true">
      <svg
        className="check-svg"
        viewBox="0 0 52 52"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="26" cy="26" r="25" strokeWidth="2" />
        <path d="M14 27l8 8 16-16" strokeWidth="3" />
      </svg>
    </div>
  )
}

// ── Shared card shell ─────────────────────────────────────────────────────────
function Shell({ children }) {
  return (
    <div className="page">
      <main className="card" role="main">
        <header className="brand">
          <span className="brand-bolt" aria-hidden="true">⚡</span>
          <span className="brand-name">Pasarela Capibara digital</span>
        </header>
        <div className="card-body">{children}</div>
      </main>
      <p className="page-footer">Pagos QR interoperables · Regulado por ASFI</p>
    </div>
  )
}

// ── States ─────────────────────────────────────────────────────────────────────
function Loading() {
  return (
    <div className="state-center" style={{ padding: '32px 0 24px' }}>
      <div className="spinner" aria-label="Cargando" role="status" />
      <p className="state-label">Verificando pago…</p>
    </div>
  )
}

function ErrorView({ message }) {
  return (
    <div className="state-center">
      <div className="icon-ring icon-ring-error" aria-hidden="true">✕</div>
      <h1 className="state-title">No se pudo cargar</h1>
      <p className="state-body">{message}</p>
    </div>
  )
}

function Pending({ payment }) {
  return (
    <>
      <div className="amount-block">
        <p className="amount-eyebrow">Total a pagar</p>
        <p className="amount" aria-label={fmt(payment.amount, payment.currency)}>
          {fmt(payment.amount, payment.currency)}
        </p>
        <p className="amount-desc">{payment.description}</p>
      </div>

      <div className="qr-wrap" role="img" aria-label="Código QR de pago">
        {payment.qrImage
          ? <img
              src={`data:image/png;base64,${payment.qrImage}`}
              alt=""
              className="qr-img"
              draggable={false}
            />
          : <div className="qr-placeholder">QR no disponible</div>
        }
      </div>

      <div className="how-to" aria-label="Instrucciones de pago">
        <p className="how-to-title">Cómo pagar</p>
        <ol className="how-to-list">
          <li>Abrí tu app bancaria</li>
          <li>Tocá "Pagar con QR" o "Escanear"</li>
          <li>Apuntá la cámara aquí</li>
          <li>Confirmá el monto y pagá</li>
        </ol>
      </div>

      <div className="status-row" aria-live="polite" aria-atomic="true">
        <span className="pulse-dot" aria-hidden="true" />
        <span>Esperando pago…</span>
      </div>

      {payment.expiresAt && (
        <p className="expiry-note">Vence el {fmtDate(payment.expiresAt)}</p>
      )}
      <p className="interop-note">Compatible con todos los bancos de Bolivia</p>
    </>
  )
}

function Paid({ payment, redirectIn }) {
  return (
    <div className="state-center">
      <CheckMark />
      <h1 className="state-title state-title-success">¡Pago recibido!</h1>
      <p className="amount-paid" aria-label={fmt(payment.amount, payment.currency)}>
        {fmt(payment.amount, payment.currency)}
      </p>

      <dl className="receipt">
        {payment.payerName && (
          <><dt>Pagador</dt><dd>{payment.payerName}</dd></>
        )}
        {payment.voucherId && (
          <><dt>Voucher</dt><dd className="receipt-mono">{payment.voucherId}</dd></>
        )}
        {payment.paidAt && (
          <><dt>Fecha</dt><dd>{fmtDateTime(payment.paidAt)}</dd></>
        )}
        {payment.sourceBank > 0 && (
          <><dt>Banco</dt><dd>{BANK_NAMES[payment.sourceBank] ?? `Banco ${payment.sourceBank}`}</dd></>
        )}
      </dl>

      {redirectIn !== null && (
        <p className="redirect-note" aria-live="polite">
          Redirigiendo en {redirectIn}s…
        </p>
      )}
    </div>
  )
}

function Expired({ payment }) {
  return (
    <div className="state-center">
      <div className="icon-ring icon-ring-warn" aria-hidden="true">⏱</div>
      <h1 className="state-title">QR expirado</h1>
      <p className="amount-muted">{fmt(payment.amount, payment.currency)}</p>
      <p className="state-body">
        Este código venció el {fmtDate(payment.expiresAt)}.{' '}
        Contactá al comercio para un nuevo enlace de pago.
      </p>
    </div>
  )
}

function Cancelled({ payment }) {
  return (
    <div className="state-center">
      <div className="icon-ring icon-ring-error" aria-hidden="true">✕</div>
      <h1 className="state-title">Pago cancelado</h1>
      <p className="amount-muted">{fmt(payment.amount, payment.currency)}</p>
      <p className="state-body">
        Este pago fue cancelado. Contactá al comercio si necesitás ayuda.
      </p>
    </div>
  )
}

// ── Root ───────────────────────────────────────────────────────────────────────
export default function PaymentPage() {
  const paymentId = window.location.pathname.replace(/^\/pay\//, '').split(/[?#]/)[0]

  const [payment, setPayment] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState(null)
  const [redirectIn, setRedirectIn] = useState(null)
  const intervalRef = useRef(null)

  useEffect(() => {
    if (!paymentId) {
      setError('URL de pago inválida.')
      setLoading(false)
      return
    }

    let active = true

    async function poll() {
      try {
        const res = await fetch(`/api/payments/${paymentId}`)
        if (!res.ok) {
          const body = await res.json().catch(() => ({}))
          throw new Error(body.error ?? `Error ${res.status}`)
        }
        const data = await res.json()
        if (!active) return

        setPayment(data)
        setLoading(false)

        if (data.status !== 'pending') {
          clearInterval(intervalRef.current)
          intervalRef.current = null
          if (data.status === 'paid' && data.callbackUrl) {
            setRedirectIn(5)
          }
        }
      } catch (e) {
        if (!active) return
        setLoading(false)
        setError(e.message ?? 'Error de conexión. Verificá la URL.')
        clearInterval(intervalRef.current)
      }
    }

    poll()
    intervalRef.current = setInterval(poll, 3000)

    return () => {
      active = false
      clearInterval(intervalRef.current)
    }
  }, [paymentId])

  // Countdown → redirect
  useEffect(() => {
    if (redirectIn === null) return
    if (redirectIn <= 0) {
      window.location.href = payment.callbackUrl
      return
    }
    const t = setTimeout(() => setRedirectIn(n => n - 1), 1000)
    return () => clearTimeout(t)
  }, [redirectIn, payment])

  return (
    <Shell>
      {loading && <Loading />}
      {!loading && error    && <ErrorView message={error} />}
      {!loading && !error   && payment?.status === 'pending'   && <Pending payment={payment} />}
      {!loading && !error   && payment?.status === 'paid'      && <Paid payment={payment} redirectIn={redirectIn} />}
      {!loading && !error   && payment?.status === 'expired'   && <Expired payment={payment} />}
      {!loading && !error   && payment?.status === 'cancelled' && <Cancelled payment={payment} />}
    </Shell>
  )
}
