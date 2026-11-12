# Relances automatiques par email — configuration self-hosters

La fonctionnalité « relances automatiques par email » (envoi de relances sur devis/factures impayées) est **opt-in** : elle est désactivée par défaut et doit être activée explicitement par un administrateur via le flag `Relance-T1` (voir [Relance-T1](#flag-relance-t1)). Tant que le flag n’est pas activé, aucune relance n’est envoyée et aucune configuration OAuth/SMTP n’est requise.

Principe fondamental : **l’envoi se fait toujours depuis le compte email du propriétaire du devis** (le compte du micro-entrepreneur / de la société émettrice), jamais depuis un compte générique de l’instance. L’instance héberge le logiciel, elle n’envoie jamais de mail à la place de ses utilisateurs.

## 1. Flag Relance-T1 (opt-in)

- Par défaut : désactivé.
- Activation : par un administrateur, via le panneau d’administration (flag de fonctionnalité `relance_email`).
- Effet : expose les écrans de configuration OAuth/SMTP et autorise le moteur de règles (Relance-T6) à émettre.

Sans activation, les routes et écrans liés sont invisibles pour les utilisateurs finaux.

## 2. OAuth Google — Gmail API (`gmail.send`)

### 2.1 Créer l’app OAuth côté Google Cloud

1. Console Google Cloud : https://console.cloud.google.com/
2. Créer un projet (ou en sélectionner un).
3. Activer l’API **Gmail API** (APIs & Services → Library → Gmail API → Enable).
4. Écran de consentement OAuth (APIs & Services → OAuth consent screen) :
   - Type d’utilisateur : **Externe** (ou Interne si workspace Google).
   - Remplir nom de l’app, email de support, domaines autorisés.
   - Ajouter le scope : `https://www.googleapis.com/auth/gmail.send`.
5. Identifiants (APIs & Services → Credentials → Create credentials → OAuth client ID) :
   - Type : **Web application**.
   - Origines JavaScript autorisées : l’URL publique de votre instance (ex. `https://devis.example.org`).
   - URI de redirection autorisés : `https://devis.example.org/api/relance/oauth/google/callback` (adapter au chemin réel de votre déploiement).
6. Noter le **Client ID** et le **Client Secret**.

### 2.2 Renseigner les credentials côté serveur

Variables d’environnement (ou secrets montés) sur l’instance :

```
GOOGLE_OAUTH_CLIENT_ID=...
GOOGLE_OAUTH_CLIENT_SECRET=...
GOOGLE_OAUTH_REDIRECT_URI=https://devis.example.org/api/relance/oauth/google/callback
```

Le Client Secret n’est jamais exposé au frontend : il reste côté serveur et sert à l’échange du code d’autorisation contre un refresh token, qui est ensuite chiffré au repos (Relance-T5, généralisation de `paymentsecrets` : AES-256-GCM, clé `ENCRYPTION_KEY`).

### 2.3 Côté utilisateur

Chaque propriétaire connecte son propre compte Google via le bouton « Connecter Gmail » : consentement OAuth, puis stockage chiffré du refresh token rattaché à son compte utilisateur. L’envoi se fait avec son adresse comme expéditeur.

## 3. OAuth Microsoft — Graph API (`Mail.Send`)

### 3.1 Créer l’app côté Microsoft Entra (Azure AD)

1. Portail Azure : https://portal.azure.com/ → Microsoft Entra ID → App registrations → New registration.
2. Nom de l’app, comptes supportés : **Accounts in any organizational directory and personal Microsoft accounts** (ou restreindre selon besoin).
3. URI de redirection : Web → `https://devis.example.org/api/relance/oauth/microsoft/callback`.
4. Dans l’app créée :
   - API permissions → Add a permission → Microsoft Graph → **Delegated permissions** → `Mail.Send` (et `offline_access` pour le refresh token).
   - Accorder le consentement administrateur si nécessaire (sinon consentement utilisateur à la première connexion).
5. Certificates & secrets → New client secret → noter la valeur (affichée une seule fois).
6. Noter l’**Application (client) ID** et le **Directory (tenant) ID** (ou `common` pour multi-tenant).

### 3.2 Renseigner les credentials côté serveur

```
MICROSOFT_OAUTH_CLIENT_ID=...
MICROSOFT_OAUTH_CLIENT_SECRET=...
MICROSOFT_OAUTH_TENANT=common
MICROSOFT_OAUTH_REDIRECT_URI=https://devis.example.org/api/relance/oauth/microsoft/callback
```

Même traitement que Google : échange de code, refresh token chiffré au repos, rattaché à l’utilisateur.

## 4. Compte SMTP manuel (côté utilisateur)

Pour les utilisateurs qui ne passent pas par OAuth (Gmail/Outlook personnels, serveur mail d’entreprise, etc.), la configuration se fait **côté utilisateur**, dans son profil :

| Champ | Exemple | Notes |
|-------|---------|-------|
| Hôte SMTP | `smtp.example.org` | Serveur sortant |
| Port | `587` (STARTTLS) ou `465` (TLS implicite) | |
| TLS | STARTTLS / TLS implicite / aucun | Privilégier TLS |
| Utilisateur | `moi@example.org` | Généralement l’adresse d’expédition |
| Mot de passe d’application | `xxxx xxxx xxxx xxxx` | Mot de passe d’application, **jamais** le mot de passe principal du compte |
| Adresse d’expédition | `moi@example.org` | Doit correspondre au compte authentifié |

Le mot de passe est chiffré au repos (Relance-T5). L’envoi se fait depuis cette adresse, qui est celle du propriétaire : cohérent avec le principe « jamais depuis un compte générique de l’instance ».

Recommandations :
- Créer un **mot de passe d’application** dédié (Gmail, Microsoft 365, etc.) plutôt qu’utiliser le mot de passe du compte.
- Vérifier que le serveur autorise l’envoi depuis l’adresse configurée (SPF/DKIM côté domaine si envoi depuis un domaine propre).
- Tester la connexion (bouton « Tester ») avant d’activer des règles de relance.

## 5. Rappels importants

- **Opt-in** : flag `Relance-T1` désactivé par défaut ; aucune relance sans activation explicite.
- **Expéditeur = propriétaire** : chaque mail part de l’adresse email du propriétaire du devis, via ses propres credentials (OAuth ou SMTP). L’instance ne possède aucun compte d’envoi générique et n’envoie jamais de mail à la place d’un utilisateur.
- **Secrets** : Client Secrets OAuth et mots de passe SMTP sont chiffrés au repos avec `ENCRYPTION_KEY` ; ne jamais les committer ni les logger.
- **Scopes minimaux** : `gmail.send` (Google) et `Mail.Send` (Microsoft) — aucun accès en lecture aux boîtes mail.
- **Désactivation** : couper le flag `Relance-T1` arrête l’émission de nouvelles relances ; les credentials restent stockés chiffrés et peuvent être supprimés par l’utilisateur.
