# mensajero

Manda los mails transaccionales de todos los sistemas de GPE. Recibe `POST /v1/email`, renderiza una
plantilla de este repo y se la entrega a Postfix (`postfix.mail.svc.cluster.local:587`), que la manda
por Amazon SES. Diseño: `gas-insideht-doc/docs/superpowers/specs/2026-10-07-mensajero-email-design.md`.

## Usarlo desde TS

```json
"mensajero": "github:GPE-Sistemas/mensajero#v1.0.0"
```

```ts
import { Mensajero, MensajeroError } from 'mensajero';

const mail = new Mensajero<'gas'>({ url: env.MENSAJERO_URL, apikey: env.MENSAJERO_APIKEY });
await mail.enviar('reset-password', { para: usuario.email, datos: { link } });
```

`MensajeroError.status`: 400 (pedido mal armado), 401 (apikey), 403 (remitente), 429 (tope por
destinatario), 502 (Postfix no respondió: se puede reintentar), 0 (no se llegó al mensajero).

## Agregar o cambiar una plantilla

1. `plantillas/<sistema>/<nombre>/`: `asunto.txt`, `cuerpo.html` (`{{define "contenido"}}...{{end}}`),
   `cuerpo.txt` (opcional) y `ejemplo.json` con exactamente los campos que usa.
2. Sólo `{{.campo}}` de primer nivel e `{{if}}`. Los datos son strings y se escapan solos.
3. `make vista-previa P=<sistema>/<nombre>` para verla.
4. `make generar` para regenerar los tipos del cliente, y `make test`.
5. Versión nueva (tag `vX.Y.Z`, y el mismo número en `package.json`): los sistemas suben de versión
   cuando quieren, y si un campo cambió, no compilan hasta adaptarse.

## Agregar un sistema

1. Su entrada en `sistemas.json` con sus remitentes (sólo `@gpe.ar` o `@horatech.ar`).
2. Su `base.html` y sus plantillas.
3. Una apikey: el hash va en el secreto del mensajero (`CLAVES`), la apikey en claro en el del sistema.
   Ver `cluster-deploy/cluster-horatech/platform/mail/`.

## Correr

| Env | Default | |
|---|---|---|
| `CLAVES` | — | `{"<sistema>": "<sha256 hex de la apikey>"}` |
| `SMTP_ADDR` | `postfix.mail.svc.cluster.local:587` | |
| `TOPE_POR_DESTINATARIO_HORA` | `30` | sumando todos los sistemas |
| `PUERTO` | `8080` | |

Logs en JSON: una línea por envío con sistema, plantilla, destinatario y message-id. Nunca los datos.
