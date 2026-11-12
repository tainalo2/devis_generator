import { useEffect, useState } from "react"

import { PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { Spinner } from "@/components/ui/spinner"
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts"

type MonthlyCA = { month: string; ca: number }
type StatusSlice = { name: string; value: number }
type Analytics = {
  monthly_ca: MonthlyCA[]
  conversion_rate: number
  by_status: StatusSlice[]
}

const MOCK: Analytics = {
  monthly_ca: [
    { month: "Jan", ca: 1200 },
    { month: "Fév", ca: 1900 },
    { month: "Mar", ca: 1600 },
    { month: "Avr", ca: 2400 },
    { month: "Mai", ca: 2100 },
    { month: "Juin", ca: 2800 },
  ],
  conversion_rate: 0.42,
  by_status: [
    { name: "Brouillon", value: 5 },
    { name: "Envoyé", value: 8 },
    { name: "Accepté", value: 6 },
    { name: "Payé", value: 12 },
    { name: "Refusé", value: 2 },
  ],
}

const COLORS = ["#6366f1", "#22c55e", "#f59e0b", "#06b6d4", "#ef4444", "#a855f7"]

export default function AnalyticsPage() {
  const [data, setData] = useState<Analytics | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch("/api/v1/analytics/overview", { credentials: "include" })
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const json = (await res.json()) as Analytics
        if (!cancelled) setData(json)
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Erreur")
          setData(MOCK)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const d = data ?? MOCK

  return (
    <>
      <Seo title="Analytics" />
      <PageHeader title="Analytics" description="Métriques de votre activité." />
      {loading ? (
        <div className="flex justify-center py-10">
          <Spinner />
        </div>
      ) : (
        <div className="grid gap-6">
          {error && (
            <p className="text-sm text-muted-foreground">
              Endpoints analytics indisponibles ({error}) — données démonstration affichées.
            </p>
          )}
          <div className="rounded-lg border p-4">
            <h2 className="mb-4 text-lg font-semibold">CA par mois</h2>
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={d.monthly_ca}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="month" />
                  <YAxis />
                  <Tooltip />
                  <Line type="monotone" dataKey="ca" stroke="#6366f1" strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
            <div className="rounded-lg border p-4">
              <h2 className="mb-4 text-lg font-semibold">Taux de conversion</h2>
              <p className="text-4xl font-bold">{(d.conversion_rate * 100).toFixed(1)} %</p>
              <p className="text-sm text-muted-foreground">Devis → facture</p>
            </div>
            <div className="rounded-lg border p-4">
              <h2 className="mb-4 text-lg font-semibold">Répartition par statut</h2>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={d.by_status} dataKey="value" nameKey="name" outerRadius={80} label>
                      {d.by_status.map((_, i) => (
                        <Cell key={i} fill={COLORS[i % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  )
}