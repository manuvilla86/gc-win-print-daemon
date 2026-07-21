# PrintBridge

Servicio local para Windows que actúa como puente entre la app web y una impresora térmica. Recibe bytes ESC/POS desde el browser por HTTP y los envía a la impresora usando el driver estándar de Windows — sin WebUSB, sin cambiar drivers.

> En Mac y Linux no es necesario: WebUSB funciona directo en Chrome.

## Build

### Publicar una nueva versión (recomendado)

El workflow de GitHub Actions compila y publica el `.exe` automáticamente al crear un tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Eso dispara [`.github/workflows/release.yml`](.github/workflows/release.yml), que cross-compila el `.exe` y lo adjunta al GitHub Release. El binario queda disponible en:

```text
https://github.com/manuvilla86/gc-win-print-daemon/releases/latest/download/PrintBridge-Setup.exe
```

### Build local

Requiere Go 1.21+.

```bash
go mod tidy
make build        # genera printbridge.exe (Windows amd64, sin ventana)
```

## Instalación

1. Descargar `PrintBridge-Setup.exe` y ejecutarlo
2. Seguir el asistente de instalación (Welcome → Install → Finish)
3. Opcionalmente marcar "Iniciar PrintBridge ahora" en la pantalla final

El instalador:

- Copia el ejecutable a `%LOCALAPPDATA%\Programs\PrintBridge\`
- Registra el startup en `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- Registra la entrada en "Agregar o quitar programas"

No requiere permisos de administrador. Para desinstalar, usar "Agregar o quitar programas" o el botón de desinstalación en la app web.

## API

Escucha en `http://localhost:9100`. Todos los endpoints responden con `Content-Type: application/json` y soportan CORS desde cualquier origen.

Ver contrato completo en [`openapi.yaml`](openapi.yaml).

---

### `GET /health`

Estado del bridge y de la impresora activa.

```json
{
  "status": "ok",
  "printer": "EPSON TM-T20III",
  "ready": true,
  "configured": true
}
```

| Campo | Descripción |
| --- | --- |
| `printer` | Nombre de la impresora activa. `null` si no hay ninguna disponible. |
| `ready` | `true` si la impresora está instalada y disponible en Windows. |
| `configured` | `true` si el usuario eligió explícitamente una impresora vía `PUT /config`. `false` si se está usando la primera detectada automáticamente. |

---

### `GET /printers`

Lista todas las impresoras instaladas en el sistema.

```json
{
  "printers": ["EPSON TM-T20III", "Microsoft Print to PDF"]
}
```

---

### `GET /config`

Devuelve la configuración guardada.

```json
{ "printer": "EPSON TM-T20III" }
```

`printer` es `""` si el usuario aún no configuró una impresora.

---

### `PUT /config`

Guarda la impresora seleccionada. La config persiste en `config.json` junto al ejecutable.

**Request:**

```json
{ "printer": "EPSON TM-T20III" }
```

**Response 200:**

```json
{ "ok": true }
```

---

### `POST /print`

Envía bytes ESC/POS crudos a la impresora activa. Usa la impresora configurada vía `PUT /config`; si no hay ninguna configurada, usa la primera detectada automáticamente.

**Request:**

- `Content-Type: application/octet-stream`
- Body: bytes ESC/POS (`Uint8Array` desde el browser)

**Response 200:**

```json
{ "ok": true }
```

**Response 503** — impresora no encontrada:

```json
{ "ok": false, "error": "PRINTER_NOT_FOUND" }
```

**Response 500** — error al imprimir:

```json
{ "ok": false, "error": "PRINT_FAILED", "detail": "..." }
```

---

## Integración con la app web

### Detectar el bridge

```typescript
const res = await fetch('http://localhost:9100/health').catch(() => null)
if (!res) {
  // bridge no instalado → mostrar link de descarga
}
const { ready, configured } = await res.json()
```

| Estado | UI sugerida |
| --- | --- |
| Bridge no responde | "App de impresión no instalada" + link de descarga |
| `ready: false`, `configured: false` | "No se detectó ninguna impresora" |
| `ready: false`, `configured: true` | "Impresora configurada no disponible — ¿está conectada?" |
| `ready: true`, `configured: false` | "Impresora detectada" + botón "Configurar" |
| `ready: true`, `configured: true` | "Lista para imprimir" |

### Configurar impresora

```typescript
const { printers } = await fetch('http://localhost:9100/printers').then(r => r.json())
// mostrar dropdown con `printers`

await fetch('http://localhost:9100/config', {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ printer: selectedPrinter }),
})
```

### Imprimir

```typescript
await fetch('http://localhost:9100/print', {
  method: 'POST',
  headers: { 'Content-Type': 'application/octet-stream' },
  body: escPosBytes, // Uint8Array
})
```
