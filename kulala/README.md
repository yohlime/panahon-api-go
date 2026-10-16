# Panahon API scratch requests

HTTP requests for the Panahon API, for the [kulala.nvim](https://github.com/mistweaverco/kulala.nvim)
HTTP client.

## Usage

- Select an environment in kulala: `dev` (default) targets `http://localhost:3000/api/v1`;
  `prod` targets the deployed API.
- Run the **login** request in `auth.http` first. The other files reuse its session cookie via
  `{{login.response.cookies.access_token.value}}`.

| File | Covers |
|------|--------|
| `auth.http` | login, current user, renew, logout |
| `users.http` | user CRUD |
| `roles.http` | roles (read: admin, write: superadmin) |
| `access.http` | per-user entitlement + access requests |
| `tokens.http` | API keys (self or admin) |
| `stations.http` | stations and station observations |
| `observations.http` | latest and bulk observations |
| `sms.http` | Lufft SMS ingress (public) |
| `poteka.http` | external Poteka API |

## Secrets

`http-client.env.json` (committed) holds only non-secret defaults. Secrets live in
`http-client.private.env.json`, which is gitignored:

| Key | Used by |
|-----|---------|
| `ADMIN_USER` | `auth.http` login — prod only; `dev` defaults to `admin` in the shared env |
| `ADMIN_PASSWORD` | `auth.http` login |
| `API_KEY` | `?api_key=` requests (`stations`, `observations`) |
| `POTEKA_API_KEY` | `poteka.http` |

Create your own `http-client.private.env.json`:

```json
{
  "dev": { "ADMIN_PASSWORD": "password", "API_KEY": "mo-..." },
  "prod": { "ADMIN_USER": "you", "ADMIN_PASSWORD": "...", "API_KEY": "mo-...", "POTEKA_API_KEY": "..." }
}
```
