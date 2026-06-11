# Print Bridge — Spec

## Qué es

Servicio local para Windows que actúa como puente entre la app web y una impresora térmica USB. Permite imprimir recibos ESC/POS desde el browser sin necesidad de WebUSB ni cambiar drivers del sistema.

## Por qué existe

WebUSB en Windows requiere reemplazar el driver de la impresora con WinUSB (via Zadig), lo cual es complejo para usuarios no técnicos. Este servicio elimina ese requisito: corre en la PC del usuario, recibe los bytes ESC/POS desde la app web por HTTP, y los envía a la impresora usando el driver estándar de Windows.

En Mac y Linux no es necesario — WebUSB funciona directo en Chrome.

## Comportamiento esperado

- Se instala una vez en la PC con Windows
- Arranca automáticamente con Windows (servicio o entrada en startup)
- Corre en segundo plano, sin ventana, sin tray icon
- Escucha en `http://localhost:9100`
- La app web detecta si está corriendo y muestra el estado al usuario

## API

### `GET /health`

Verifica que el servicio está corriendo e indica si hay una impresora detectada.

**Response 200:**
```json
{
  "status": "ok",
  "printer": "EPSON TM-T20III",   // nombre del dispositivo, o null si no hay impresora
  "ready": true                    // true si hay impresora lista para imprimir
}
```

**Response 200 (sin impresora):**
```json
{
  "status": "ok",
  "printer": null,
  "ready": false
}
```

### `POST /print`

Recibe los bytes ESC/POS y los envía a la impresora.

**Request:**
- Content-Type: `application/octet-stream`
- Body: bytes ESC/POS crudos (Uint8Array desde el browser)

**Response 200:**
```json
{ "ok": true }
```

**Response 503:**
```json
{ "ok": false, "error": "PRINTER_NOT_FOUND" }
```

**Response 500:**
```json
{ "ok": false, "error": "PRINT_FAILED", "detail": "..." }
```

## Integración con la app web

La app web detecta el bridge con un `fetch('http://localhost:9100/health')`. Según la respuesta:

| Estado | UI |
|--------|-----|
| Bridge no responde | "App de impresión no instalada" + link de descarga |
| Bridge responde, `ready: false` | "Impresora no conectada" |
| Bridge responde, `ready: true` | "Lista para imprimir" (verde) |

Al imprimir, la app hace:
```typescript
const res = await fetch('http://localhost:9100/print', {
  method: 'POST',
  headers: { 'Content-Type': 'application/octet-stream' },
  body: escPosBytes,
})
```

El bridge maneja CORS para permitir requests desde cualquier origen local.

## Detección de impresora

El bridge detecta automáticamente la primera impresora USB disponible en el sistema. No requiere configuración manual del nombre o puerto — usa la API de Windows para enumerar impresoras y selecciona la primera que responda.

Si hay múltiples impresoras, usa la primera de la lista. En versiones futuras se podría agregar configuración para seleccionar cuál usar.

## Tecnología recomendada

**Go** — binario standalone sin dependencias, ~5MB, compatible con Windows 7+.

Librerías:
- `net/http` — servidor HTTP (stdlib)
- `golang.org/x/sys/windows` — acceso a API de Windows para impresoras
- Compilado con `GOARCH=amd64 GOOS=windows`

## Distribución

1. El `.exe` se publica como GitHub Release en un repo separado
2. Desde la app web, en **Configuración → Sistema**, hay un botón "Descargar app de impresión" que apunta a la URL del último release
3. El usuario descarga y ejecuta el `.exe` — Windows puede mostrar SmartScreen la primera vez, el usuario hace click en "Más información" → "Ejecutar de todas formas"
4. El servicio queda instalado y arranca con Windows

## Instalación en Windows

Al ejecutar el `.exe` por primera vez:
1. Se copia a `C:\Program Files\PrintBridge\`
2. Se registra en `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` para arrancar con Windows
3. Comienza a escuchar en `localhost:9100`

No requiere permisos de administrador.

## Futuras mejoras (fuera de scope inicial)

- Selección de impresora cuando hay múltiples
- Auto-actualización
- Soporte para impresoras de red (TCP/IP) además de USB
- Firma de código para eliminar el warning de SmartScreen
