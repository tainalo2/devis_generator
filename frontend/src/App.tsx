import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"

import { AppLayout } from "@/components/app-layout"
import { AuthProvider } from "@/components/auth-provider"
import { ThemeProvider } from "@/components/theme-provider"
import { AccountPage } from "@/pages/account-page"
import { ClientEditPage } from "@/pages/client-edit-page"
import { ClientsPage } from "@/pages/clients-page"
import { DocumentDetailPage } from "@/pages/document-detail-page"
import { DocumentsPage } from "@/pages/documents-page"
import { HomePage } from "@/pages/home-page"
import { IssuerPage } from "@/pages/issuer-page"
import { LegalSourcesPage } from "@/pages/legal-sources-page"
import { LoginPage } from "@/pages/login-page"
import { NewDocumentPage } from "@/pages/new-document-page"
import { RegisterPage } from "@/pages/register-page"

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route index element={<HomePage />} />
              <Route path="documents" element={<DocumentsPage />} />
              <Route path="documents/nouveau" element={<NewDocumentPage />} />
              <Route path="documents/:id" element={<DocumentDetailPage />} />
              <Route path="clients" element={<ClientsPage />} />
              <Route path="clients/nouveau" element={<ClientEditPage />} />
              <Route path="clients/:id" element={<ClientEditPage />} />
              <Route path="entreprise" element={<IssuerPage />} />
              <Route path="sources-legales" element={<LegalSourcesPage />} />
              <Route path="connexion" element={<LoginPage />} />
              <Route path="inscription" element={<RegisterPage />} />
              <Route path="compte" element={<AccountPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  )
}
