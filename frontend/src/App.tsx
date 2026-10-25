import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"

import { AppShell } from "@/components/app-shell"
import { AuthProvider } from "@/components/auth-provider"
import { PublicLayout } from "@/components/public-layout"
import { ThemeProvider } from "@/components/theme-provider"
import { ToastProvider } from "@/components/toast-provider"
import { AccountPage } from "@/pages/account-page"
import { ClientEditPage } from "@/pages/client-edit-page"
import { ClientsPage } from "@/pages/clients-page"
import { DashboardPage } from "@/pages/dashboard-page"
import { DocumentDetailPage } from "@/pages/document-detail-page"
import { DocumentsPage } from "@/pages/documents-page"
import { IssuerPage } from "@/pages/issuer-page"
import { LandingPage } from "@/pages/landing-page"
import { LegalSourcesPage } from "@/pages/legal-sources-page"
import { LoginPage } from "@/pages/login-page"
import { MentionsLegalesPage } from "@/pages/mentions-legales-page"
import { NewDocumentPage } from "@/pages/new-document-page"
import { NotFoundPage } from "@/pages/not-found-page"
import { PrivacyPage } from "@/pages/privacy-page"
import { RegisterPage } from "@/pages/register-page"

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <ToastProvider>
          <BrowserRouter>
            <Routes>
              <Route element={<PublicLayout />}>
                <Route index element={<LandingPage />} />
                <Route path="mentions-legales" element={<MentionsLegalesPage />} />
                <Route path="confidentialite" element={<PrivacyPage />} />
                <Route path="connexion" element={<LoginPage />} />
                <Route path="inscription" element={<RegisterPage />} />
                <Route path="*" element={<NotFoundPage />} />
              </Route>

              <Route element={<AppShell />}>
                <Route path="tableau-de-bord" element={<DashboardPage />} />
                <Route path="documents" element={<DocumentsPage />} />
                <Route path="documents/nouveau" element={<NewDocumentPage />} />
                <Route path="documents/:id" element={<DocumentDetailPage />} />
                <Route path="clients" element={<ClientsPage />} />
                <Route path="clients/nouveau" element={<ClientEditPage />} />
                <Route path="clients/:id" element={<ClientEditPage />} />
                <Route path="entreprise" element={<IssuerPage />} />
                <Route path="sources-legales" element={<LegalSourcesPage />} />
                <Route path="compte" element={<AccountPage />} />
                <Route path="accueil" element={<Navigate to="/" replace />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </ToastProvider>
      </AuthProvider>
    </ThemeProvider>
  )
}
