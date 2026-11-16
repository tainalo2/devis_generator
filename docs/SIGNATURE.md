# Guide de déploiement — Signature électronique simple

Documentation complète pour déployer et gérer la signature électronique simple (eIDAS art. 3(10)) : concepts juridiques, limites, preuves fournies, et configuration optionnelle pour Yousign.

**⚠️ Disclaimer** : Ce document décrit un **mécanisme technique**. Il n'offre aucune garantie juridique. Les implications légales d'une signature dépendent du contexte, de la juridiction et de la reconnaissance requise. Consultez un professionnel du droit pour les usages critiques.

## Table des matières

1. [Qu'est-ce qu'une signature électronique simple eIDAS](#1-quest-ce-quune-signature-électronique-simple-eidas)
2. [Limites juridiques et valeur probante](#2-limites-juridiques-et-valeur-probante)
3. [Ce que le mécanisme prouve](#3-ce-que-le-mécanisme-prouve)
4. [Ce que le mécanisme ne prouve pas](#4-ce-que-le-mécanisme-ne-prouve-pas)
5. [Modèle de données et stockage](#5-modèle-de-données-et-stockage)
6. [Configuration de base (Devis Generator interne)](#6-configuration-de-base-devis-generator-interne)
7. [Configuration tierce — Yousign BYO](#7-configuration-tierce--yousign-byo)
8. [Vérification et audit](#8-vérification-et-audit)
9. [Incidents et révocation](#9-incidents-et-révocation)

---

## 1. Qu'est-ce qu'une signature électronique simple eIDAS

### Définition légale

Une **signature électronique simple** au sens du [Règlement UE 910/2014 (eIDAS)](https://eur-lex.europa.eu/legal-content/FR/TXT/?uri=CELEX:32014R0910), article 3(10) est :

> Données sous forme électronique, annexées ou logiquement associées à d'autres données électroniques et utilisées par le signataire pour consentir.

**Caractéristiques clés** :
- ✓ Peut être créée par n'importe quel moyen technique
- ✓ Ne requiert pas de tiers de confiance
- ✓ Pas de certificat numérique requis
- ✓ Peut être simplement un hash + horodatage

**Ce qu'elle n'est PAS** :
- ✗ Pas une **signature avancée** (eIDAS art. 26) → nécessite certificat et horodatage tiers
- ✗ Pas une **signature qualifiée** (eIDAS art. 3(12)) → nécessite certificat qualifié + horodatage qualifié
- ✗ Pas équivalente juridiquement à une signature manuscrite (sauf cas particuliers)

### Équivalence juridique

En **France**, une signature électronique simple est :
- **Acceptée** comme preuve devant les tribunaux (Code Civil, art. 1316-1) **si elle identifie l'auteur et traduit son intention**.
- **Pas d'équivalence automatique** avec une signature manuscrite : c'est une preuve parmi d'autres, à évaluer cas par cas.
- **Poids probant laissé au juge** : en cas de litige, la cour examine si la signature assure l'intégrité et l'authenticité de l'acte.

### Niveaux de confiance eIDAS (non pertinent pour V1)

eIDAS définit trois niveaux de signature électronique :
1. **Simple** (art. 3(10)) : aucune exigence technique formelle ← **Devis Generator V1**
2. **Avancée** (art. 26) : identifie l'auteur + horodatage tiers indépendant
3. **Qualifiée** (art. 3(12)) : certificat qualifié + preuve non-répudiation

Devis Generator implémente le **niveau 1 (simple)** en V1.

---

## 2. Limites juridiques et valeur probante

### Acceptabilité selon le contexte

| Contexte | Acceptée ? | Niveau de risque | Notes |
|----------|-----------|-----------------|-------|
| Devis entre TPE/micro-entrepreneurs | ✓ Probablement | Faible | Usages courants, parties consentantes, pas de litiges graves attendus |
| Facture commerciale B2B PME | ✓ Probablement | Faible-Moyen | Obligation légale en FR pour facture électronique, mais signature simple suffit |
| Contrat à fort enjeu (prêt, partenariat) | ✗ Non recommandé | Élevé | Signature qualifiée préférable ; simple sera contestée en cas de litige |
| Acte notarié / administratif | ✗ Non acceptée | N/A | Signature qualifiée obligatoire par la loi |
| Documents sensibles (données médicales) | ✗ Non recommandé | Critique | Conformité RGPD et légale exigent traçabilité plus forte |

### Risques identifiés

1. **Pas de non-répudiation juridique** : un signataire peut contester : *« Ce n'était pas moi »* → la signature ne le prouve pas definitively.
2. **Dépendance à l'IP/session** : une IP partagée (bureau, VPN) ou session hijackée invalide la preuve d'identité.
3. **Pas de tiers indépendant** : il n'y a que vous (auto-hébergé) ou Devis Generator pour témoigner. En cas de suspicion de manipulation, aucun tiers neutre.
4. **Absence de PKI (clé publique/privée)** : pas de crypto-preuve que c'est le signataire qui a généré la signature.
5. **Recevabilité judiciaire incertaine** : même si juridiquement acceptée, un défendeur peut la contester ; c'est alors au juge de trancher.

### Droit du travail / Situations spéciales

- **Bulletins de paie** : signature simple peut suffire si contexte de confiance établi.
- **Mandats SEPA** : signature qualifiée **exigée** par PSD2 / droit bancaire (hors scope Devis Generator).
- **Courrier électronique simple** : signature simple suffit ; la reconnaissance technique (timestamp, hash) + authenticité du compte suffisent.

---

## 3. Ce que le mécanisme prouve

### Horodatage (Timestamp)

**Enregistrement** : `signed_at` = timestamp serveur au moment du clic.

**Preuves** :
- ✓ Le document a été signé à cette date/heure spécifique.
- ✓ Irréfutable a posteriori via audit logs serveur (immuable si stocké avec intégrité).
- ✓ Accepté comme preuve d'antériorité (ex : « la facture existait avant la date de litige »).

**Limites** :
- Dépend de l'horloge serveur (NTP synchronisation recommandée en prod).
- Si audit logs sont altérés, timestamp ne tient plus.

### Intégrité du document (Hash SHA-256)

**Enregistrement** : `document_hash` = SHA-256(contenu du document) au moment de la signature.

**Preuves** :
- ✓ Le contenu signé n'a pas changé (reproduire le hash = même contenu).
- ✓ Toute modification ultérieure du document change le hash → discordance = document modifié après signature.
- ✓ Accepté comme preuve technique d'intégrité (cryptographiquement très difficile à forger).

**Limites** :
- Exige que le document original soit conservé (sinon pas moyen de reproduire le hash).
- Hash seul ≠ preuve qu'on a lu/accepté le contenu (hasard ou contrainte ?).
- Si document est chiffré, hash du chiffré ≠ hash du contenu (gérer : chaîner chiffrement ↔ hash).

### Traçabilité technique (IP, User-Agent, Nom déclaré)

**Enregistrement** : `signer_ip`, `signer_user_agent`, `signer_name_declared`.

**Preuves** :
- ✓ Le signataire a accédé depuis cette IP, avec ce navigateur, à cette date.
- ✓ Permet traçabilité technique en cas d'investigation ultérieure.
- ✓ Décourage les abus : traçabilité techniquement immédiate crée responsabilité perçue.

**Limites** :
- IP partagée (bureau ouvert, WiFi public) : ne prouve pas qui en personne.
- User-agent usurpable (très facile en frontend ; validation backend nécessaire).
- Nom déclaré : simple chaîne de caractères, **aucune vérification d'identité** (voir [§ 4](#4-ce-que-le-mécanisme-ne-prouve-pas)).

### Authentification du compte

**En mode compte authentifié** :
- ✓ Signature liée à un compte utilisateur identifié (email/mot de passe ou SSO).
- ✓ Fournit une couche supplémentaire d'imputabilité : « c'est ce compte qui a signé ».

**En mode invité** (sans compte) :
- ✗ Pas d'authentification préalable ; seul le nom déclaré identifie.
- ⚠️ Acceptabilité diminue ; voir § 2 sur valeur probante.

---

## 4. Ce que le mécanisme ne prouve pas

### Identité vérifiée du signataire

**Limitation clé** : Devis Generator ne **vérifie pas** qui est réellement le signataire.

- **Pas d'authentification tiers** : pas de Yousign, DocuSign ou autre prestataire de confiance.
- **Pas de certificat numérique personnel** : pas de crypto-identité du signataire.
- **Pas de vérification d'email** : accepte n'importe quel nom déclaré.

**Implications** :
- Signature relie une **action technique** (clic sur ce compte/IP) à un **nom**, pas à une **personne**.
- En cas de litige : « Oui, quelqu'un a cliqué en mon IP, mais ce n'était pas moi » → argument valide sans preuve du contraire.

### Non-répudiation juridique

**Concept** : le signataire ne peut pas nier avoir signé (niveau de certitude élevé).

**Devis Generator ne le fournit pas** :
- Pas de preuve cryptographique que c'est le signataire qui a généré la signature (clé privée).
- Pas de tiers neutre pour témoigner de l'authentification.
- Pas de PKI (Public Key Infrastructure).

**Résultat** : signature prouve une **action**, pas une **intention**. Utile pour usages confiants (interne PME), dangereux pour litige potentiel.

### Session hijackée

**Scénario** : attaquant accède au compte/IP du signataire, signe un document.

**Devis Generator ne protège pas contre ça** :
- Signature enregistrerait l'IP de l'attaquant (pas le vrai signataire).
- Pas d'authentification multi-facteur obligatoire.
- Pas de tiers pour vérifier qui clique vraiment.

**Mitigation** : compter sur **sécurité du compte** (mot de passe fort, 2FA optionnelle, audit logs) + **confiance mutuelle** entre parties.

### Conformité RGPD / Données à caractère personnel

**Le mécanisme enregistre** : IP, user-agent, nom signataire déclaré.

**Implications RGPD** :
- IP = donnée à caractère personnel → traitement RGPD.
- Droit à l'oubli / rectification / accès doivent s'appliquer.
- Consentement explicite recommandé avant signature.

**Responsabilité** : chaque auto-hébergeur est responsable de sa conformité RGPD. Devis Generator fournit les données, pas le cadre légal.

---

## 5. Modèle de données et stockage

### Schéma de table

```sql
CREATE TABLE document_signatures (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  
  -- Identité signataire
  signer_name_declared VARCHAR(255) NOT NULL,  -- Nom déclaré (non vérifiée)
  signer_account_id UUID REFERENCES accounts(id),  -- NULL si invité
  
  -- Preuves
  signed_at TIMESTAMPTZ NOT NULL,                  -- Horodatage serveur
  document_hash VARCHAR(64) NOT NULL,              -- SHA-256 hex
  signer_ip INET NOT NULL,                         -- IPv4 ou IPv6
  signer_user_agent TEXT NOT NULL,                 -- HTTP User-Agent header
  
  -- Audit
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  signature_method VARCHAR(32) NOT NULL DEFAULT 'simple-eidas-v1',
  signature_version INT NOT NULL DEFAULT 1,
  
  -- Métadonnées optionnelles
  additional_context JSONB,  -- Ex : { "document_type": "facture", ... }
  
  CONSTRAINT valid_hash CHECK (LENGTH(document_hash) = 64),
  CONSTRAINT valid_method CHECK (signature_method IN ('simple-eidas-v1', 'yousign-byo-v1'))
);

CREATE INDEX idx_document_signatures_document_id ON document_signatures(document_id);
CREATE INDEX idx_document_signatures_signer_account ON document_signatures(signer_account_id);
CREATE INDEX idx_document_signatures_signed_at ON document_signatures(signed_at DESC);
```

### Champs obligatoires

| Champ | Type | Description |
|-------|------|-------------|
| `id` | UUID | Identifiant unique de la signature |
| `document_id` | UUID | Référence au document signé |
| `signer_name_declared` | String | Nom déclaré du signataire (non vérifié) |
| `signed_at` | Timestamp | Date/heure de signature (serveur) |
| `document_hash` | String[64] | Hash SHA-256 du document |
| `signer_ip` | INET | IP source du signataire |
| `signer_user_agent` | String | User-Agent HTTP du signataire |

### Champs optionnels

| Champ | Type | Description |
|-------|------|-------------|
| `signer_account_id` | UUID | Ref. au compte utilisateur (NULL = invité) |
| `additional_context` | JSONB | Métadonnées : type doc, raison signature, etc. |

### Chiffrement des données sensibles

**IP et user-agent** : peuvent révéler localisation/fingerprint du signataire.

**Recommandation** :
- Appliquer AES-256-GCM (même clé que `ENCRYPTION_KEY`) au moins à `signer_ip`.
- `signer_user_agent` : chiffrable mais moins sensible ; peut rester en clair si audit logs protégés.

**Stockage sécurisé** :
```go
// Pseudocode : chiffrer IP avant INSERT
encryptedIP := encrypt(signerIP, encryptionKey)
signature.SignerIP = encryptedIP  // stockage
// Pour audit : déchiffrer lors de la consultation
decryptedIP := decrypt(signature.SignerIP, encryptionKey)
```

### Conservation et suppression

**Durée de conservation** :
- **Défaut** : aussi longtemps que le document existe (lié par FK).
- **RGPD** : minimum 3 ans (délai de prescription en droit français) ; purge explicite après.
- **Optionnel** : job périodique de purge pour comptes supprimés (data minimization).

**Suppression du compte utilisateur** :
- Cascade : signatures d'invitées restent (pas liées à compte).
- Signatures de compte supprimé : anonymiser `signer_name_declared`, NULL `signer_account_id`, conserver horodatage/hash/IP pour traçabilité.

---

## 6. Configuration de base (Devis Generator interne)

### Variables d'environnement

```bash
# Variables de signature (toutes optionnelles en V1 car implémentation interne)
# À compléter si option BYO Yousign ajoutée

# Futur (Yousign BYO) — non utilisé en V1
SIGNATURE_PROVIDER_ENABLED=false     # Activation du provider tiers
SIGNATURE_PROVIDER=yousign           # "yousign" (seule option envisagée V2)
SIGNATURE_YOUSIGN_API_KEY=           # Clé API Yousign (fournie par utilisateur)
SIGNATURE_YOUSIGN_WORKSPACE_ID=      # Workspace Yousign (fournie par utilisateur)
SIGNATURE_HASH_ALGORITHM=sha256      # Toujours SHA-256 pour V1
```

### Comportement par défaut (V1)

| Paramètre | Valeur | Effet |
|-----------|--------|-------|
| `SIGNATURE_PROVIDER_ENABLED` | `false` | Signatures internes uniquement (via table `document_signatures`) |
| Hash | SHA-256 | Standard cryptographique actuel |
| Horodatage | Serveur (NOW()) | Dépend de l'horloge du serveur (NTP recommandé) |
| IP/User-Agent | Enregistrés | Traçabilité technique activée |
| Chiffrement IP | AES-256-GCM | Optionnel (voir § 5) |

### Vérification d'intégrité côté serveur

Avant d'enregistrer une signature, vérifier l'intégrité du document :

```go
// Pseudocode
func SignDocument(docID string, signerName string, signerIP string, r *http.Request) error {
    // 1. Récupérer le contenu du document
    docContent, err := db.GetDocumentContent(docID)
    if err != nil {
        return fmt.Errorf("document not found")
    }
    
    // 2. Calculer le hash SHA-256
    hash := sha256.Sum256([]byte(docContent))
    hashHex := hex.EncodeToString(hash[:])
    
    // 3. Enregistrer la signature
    sig := &DocumentSignature{
        DocumentID:        docID,
        SignerNameDeclared: signerName,
        SignedAt:          time.Now().UTC(),
        DocumentHash:      hashHex,
        SignerIP:          r.RemoteAddr,  // Ou récupérer IP réelle si proxy
        SignerUserAgent:    r.Header.Get("User-Agent"),
    }
    
    return db.InsertSignature(sig)
}
```

### Vérification ultérieure

Après signature, pour vérifier l'intégrité du document :

```go
// Pseudocode
func VerifyDocumentIntegrity(docID string, signatureID string) (bool, error) {
    // 1. Récupérer le contenu actuel du document
    currentContent, err := db.GetDocumentContent(docID)
    if err != nil {
        return false, err
    }
    
    // 2. Récupérer la signature
    sig, err := db.GetSignature(signatureID)
    if err != nil {
        return false, err
    }
    
    // 3. Calculer le hash actuel
    currentHash := hex.EncodeToString(sha256.Sum256([]byte(currentContent))[:])
    
    // 4. Comparer
    if currentHash == sig.DocumentHash {
        return true, nil  // Document inchangé
    }
    return false, fmt.Errorf("document integrity check failed: hash mismatch")
}
```

### Endpoint API

```http
POST /api/v1/documents/:document_id/sign
Content-Type: application/json
Authorization: Bearer <token-ou-anonyme>

{
  "signer_name": "Marie Dupont"
}

---

Response 200 OK:

{
  "signature": {
    "id": "sig_abc123xyz",
    "document_id": "doc_xyz789",
    "signer_name_declared": "Marie Dupont",
    "signed_at": "2026-11-16T18:30:45Z",
    "document_hash": "a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7",
    "signer_ip": "192.0.2.42",
    "signer_user_agent": "Mozilla/5.0 ...",
    "signature_method": "simple-eidas-v1"
  }
}
```

### Affichage côté utilisateur

Après signature, afficher un badge ou attestation :

```
✓ Signé par : Marie Dupont
  Date : 16 novembre 2026, 18h30 (UTC)
  Intégrité : vérifiée (hash SHA-256)
  Traçabilité : IP 192.0.2.42 | Mozilla/5.0 ...
  
  Statut juridique : Signature électronique simple eIDAS (V1)
  ⚠️ Non qualifiée. Pas de tiers de confiance. Consulter SIGNATURE.md.
```

---

## 7. Configuration tierce — Yousign BYO

### Concepts

**BYO (Bring Your Own) = Clé API fournie par l'utilisateur**

Modèle : même pattern que `PAYMENT_BYO_ENABLED` (paiements).

- Utilisateur crée compte Yousign (gratuit ou payant).
- Utilisateur génère clé API Yousign.
- Auto-hébergeur passe clé API en variable d'env.
- Signatures optionnellement relayées à Yousign pour horodatage tiers qualifié.

### Avantages (si Yousign ajouté en V2+)

✓ **Horodatage tiers qualifié** : Yousign fournit timestamp qualifié (eIDAS art. 26).
✓ **Non-répudiation plus forte** : tiers indépendant témoigne.
✓ **Acceptabilité juridique augmentée** : pour usages semi-critiques.
✓ **Optionnel** : reste gratuit si Yousign désactivé.

### Limitations (honnêtes)

✗ **Coût** : plan Yousign peut être facturé après période gratuite.
✗ **Dépendance externe** : Yousign doit être up ; indisponibilité = pas de signature.
✗ **Données tiers** : IP/user-agent partagés avec Yousign (implication RGPD).
✗ **Pas qualifié sans certificat** : Yousign seul (clé API) ne suffit pas pour signature qualifiée ; faut certificat personnel.
✗ **Hors scope V1** : non implémenté ; aucun besoin confirmé.

### Configuration (future V2+)

```bash
# .env
SIGNATURE_PROVIDER_ENABLED=true
SIGNATURE_PROVIDER=yousign
SIGNATURE_YOUSIGN_API_KEY=sk_live_abc123xyz789...  # Clé API Yousign
SIGNATURE_YOUSIGN_WORKSPACE_ID=ws_live_def456...   # Workspace Yousign (optionnel)
SIGNATURE_HASH_ALGORITHM=sha256                     # Toujours SHA-256
```

### Workflow (future)

1. **Utilisateur clique « Signer »** dans Devis Generator.
2. **Côté serveur** :
   - Hash SHA-256 du document → `hash_value`.
   - Envoyer à Yousign : `POST /api/signatures` avec `hash_value`, `signer_name`, clé API.
   - Yousign retourne : `signature_id`, `timestamp_qualified`, `proof_url`.
3. **Enregistrement local** :
   - Créer entry `document_signatures` : hash, IP, user-agent, **ET** `yousign_signature_id`.
   - Stocker timestamp Yousign en champ optionnel (cas où horodatage tiers > horodatage local).
4. **Vérification ultérieure** :
   - Vérifier via Yousign API si signature toujours valide.
   - Ou récupérer preuve PDF de Yousign (optionnel, pour archivage).

### Tableau de comparaison

| Aspect | Interne V1 | Yousign BYO (future) |
|--------|------------|---------------------|
| **Horodatage** | Serveur seul (non tiers) | Yousign qualifié |
| **Non-répudiation** | Faible (pas tiers) | Meilleur (tiers) |
| **Traçabilité** | IP/User-Agent locaux | IP/User-Agent + Yousign logs |
| **Coût** | Gratuit (auto-hébergement) | Gratuit ou payant (Yousign) |
| **Dépendance externe** | Aucune | Yousign (statut/uptime) |
| **Acceptabilité juridique** | Faible-Moyen (eIDAS simple) | Moyen-Élevé (tiers) |
| **Équivalence signature qualifiée** | ✗ Non | ✗ Non (pas certificat perso) |
| **Complexité implémentation** | Faible | Moyenne |

### Statut : hors V1

**Raison** : pas de besoin confirmé. Signatures simples suffisent pour TPE/micro-entrepreneurs (cas d'usage Devis Generator).

**Revisiter si** :
- Utilisateurs demandent horodatage tiers.
- Litige judiciaire montre limitations du simple.
- Conformité légale exige meilleur niveau.

---

## 8. Vérification et audit

### API de vérification

```http
GET /api/v1/documents/:document_id/signatures
Authorization: Bearer <token>

---

Response 200 OK:

{
  "document_id": "doc_xyz789",
  "signatures": [
    {
      "id": "sig_abc123xyz",
      "signer_name_declared": "Marie Dupont",
      "signed_at": "2026-11-16T18:30:45Z",
      "document_hash": "a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7",
      "hash_verified": true,  -- true si contenu actuel = hash signé
      "signature_method": "simple-eidas-v1"
    }
  ],
  "document_content_hash": "a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7",  -- Hash actuel
  "integrity_status": "verified"  -- "verified" = aucune signature n'a detecté modification
}
```

### Vérification manuelle

Auto-hébergeur peut vérifier intégrité d'un document **offline** :

```bash
# 1. Exporter le document signé (PDF, XML, etc.)
# 2. Calculer son hash
sha256sum document_signé.pdf

# Résultat : a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7

# 3. Comparer avec hash enregistré en base
# Si identique : document inchangé depuis signature
```

### Audit logs

**Chaque signature doit laisser une trace**

```sql
-- Table d'audit (append-only)
CREATE TABLE signature_audit_log (
  id BIGSERIAL PRIMARY KEY,
  signature_id UUID NOT NULL REFERENCES document_signatures(id),
  event_type VARCHAR(50) NOT NULL,  -- "created", "verified", "failed_verification", "revoked"
  event_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  actor_id UUID REFERENCES accounts(id),  -- NULL si système
  actor_role VARCHAR(50),  -- "user", "admin", "system"
  details JSONB,  -- { "reason": "...", "hash_mismatch": true, ... }
  
  CONSTRAINT valid_event_type CHECK (event_type IN ('created', 'verified', 'failed_verification', 'revoked'))
);

CREATE INDEX idx_signature_audit_signature_id ON signature_audit_log(signature_id);
CREATE INDEX idx_signature_audit_timestamp ON signature_audit_log(event_timestamp DESC);
```

### Rapports de conformité

Générer périodiquement (ex : mensuellement) :

```
=== Rapport de signature électronique — Novembre 2026 ===

Signatures créées : 1,234
Vérifications effectuées : 892
Intégrité vérifiée : 892/892 (100%)
Signatures révoquées : 0
Incidents détectés : 0

Signataires uniques : 234
Utilisateurs authentifiés : 198
Invités : 36

Documents signés (types) :
  - Factures : 456 (37%)
  - Devis : 567 (46%)
  - Avoirs : 211 (17%)

Statut juridique : Conforme eIDAS V1 (simple, non qualifié)
```

---

## 9. Incidents et révocation

### Révocation d'une signature

**Cas** : signature suspecte, contestée, ou erreur technique.

**Processus** :

1. **Identification** :
   ```sql
   SELECT * FROM document_signatures 
   WHERE id = 'sig_abc123xyz';
   ```

2. **Vérification** :
   - Hash du document actuel vs hash signé → incohérence ?
   - Audit logs : qui a créé, quand ?
   - Signataire vs compte : cohérent ?

3. **Révocation** (soft delete) :
   ```sql
   UPDATE document_signatures
   SET revoked_at = NOW(),
       revocation_reason = 'Signature conteste par signataire',
       revocation_actor_id = :admin_id
   WHERE id = 'sig_abc123xyz';
   ```

4. **Notification** :
   - Email au signataire : « Votre signature du document XYZ a été révoquée le [date]. Raison : [reason]. »
   - Email à propriétaire du document : « Signature révoquée. Document non signé. »

5. **Audit** :
   ```sql
   INSERT INTO signature_audit_log (signature_id, event_type, actor_id, details)
   VALUES ('sig_abc123xyz', 'revoked', :admin_id, 
           jsonb_build_object('reason', 'Signature conteste par signataire'));
   ```

### Incidents types

| Incident | Cause probable | Action |
|----------|----------------|--------|
| Hash mismatch | Document modifié après signature | ✓ Intégrité échouée ; signature invalide |
| IP/User-Agent suspecte | Session hijackée ou usurpation | ⚠️ Enquête ; révocation recommandée |
| Signature fantôme | Bug logiciel ou base corrompue | 🚨 Audit logs + rollback si nécessaire |
| Signataire nié | Contestation légale | ⚠️ Hors scope système ; à traiter légalement |
| Yousign indisponible | Service tiers down (si BYO actif) | ⚠️ Fallback : signature interne ; re-tenter tiers plus tard |

### Procédure en cas de litige

**L'auto-hébergeur doit conserver** :
- ✓ Document original + signature (hash, timestamp, IP, user-agent).
- ✓ Audit logs complets (table `signature_audit_log`).
- ✓ Logs d'accès serveur (HTTP access logs, optionnel mais recommandé).
- ✓ Configuration cryptographique (algorithmes, clés) au moment de la signature.

**En cas de contestation légale** :
1. Produire rapport d'intégrité : hash signé vs actuel → preuves.
2. Produire audit logs : qui a signé, quand, d'où.
3. Produire preuve de compte/IP du signataire.
4. **Reconnaître limites** : pas de preuve que c'était réellement la personne ; c'est preuve technique, pas légale.
5. Consulter avocat ; cas au juge.

### Cryptographie : rotation ou mise à jour

**Hash algorithm change** (ex : SHA-256 → SHA-3) :

1. Maintenir compatibilité **backwards** : vérifier les deux algos.
2. Nouvelles signatures utilisent SHA-3 (versionnée en DB).
3. Anciennes signatures (SHA-256) restent valides.
4. Migration progressive (optionnel) : re-signer documents anciens avec nouvel algo.

**Exemple schéma** :
```sql
ALTER TABLE document_signatures ADD COLUMN hash_algorithm VARCHAR(32) DEFAULT 'sha256';

-- Migration future
UPDATE document_signatures SET hash_algorithm = 'sha3-256' 
WHERE signature_version >= 2;
```

---

## Résumé et checklist de déploiement

### Avant déploiement en prod

- [ ] Table `document_signatures` créée + indexes.
- [ ] Chiffrement AES-256-GCM appliqué à `signer_ip` (optionnel mais recommandé).
- [ ] Vérification d'intégrité implémentée (hash SHA-256).
- [ ] API endpoints testés (`/sign`, `/signatures`, `/verify`).
- [ ] Audit logs fonctionnels.
- [ ] Badge/affichage informatif ajouté (mention "simple eIDAS, non qualifiée").
- [ ] Pages légales/Privacy Policy mises à jour (IP enregistrée).
- [ ] Logs serveur archivés sécurisé (audit trail).
- [ ] Équipe consciente des limites juridiques (voir § 2, 4).
- [ ] Lien vers ce doc (`SIGNATURE.md`) visible en UI.

### Monitoring continu

- [ ] Vérification d'intégrité montée en alerte (hash mismatches).
- [ ] Incidents de signature documentés.
- [ ] Rapports mensuels générés.
- [ ] Yousign BYO envisagé **si demande utilisateur** (future V2+).

### Communication utilisateur

- ✓ Documentation claire : « Quelle est la valeur juridique ? »
- ✓ Disclaimer : « Cette signature n'est pas qualifiée. »
- ✓ Lien vers ce document fourni.
- ✓ Email post-signature : détails + preuve d'intégrité.

---

## Ressources externes

- **eIDAS Règlement UE 910/2014** : https://eur-lex.europa.eu/legal-content/FR/TXT/?uri=CELEX:32014R0910
  - Article 3(10) : définition signature simple
  - Article 26 : signature avancée
  - Article 3(12) : signature qualifiée

- **Code Civil français (Contrats)** :
  - Article 1316-1 : acceptabilité signature électronique
  - Article 1365 et s. : immuabilité actes

- **Yousign (si intégration future)** : https://www.yousign.com/api

- **ANSSI (France)** : Recommandations cryptographiques
  - https://www.anssi.gouv.fr/

---

**Document date** : 2026-11-16
**Statut** : Référence pour Devis Generator V1
**Mise à jour suivante** : Si besoin Yousign BYO confirmé (V2+) ou incident de signature
