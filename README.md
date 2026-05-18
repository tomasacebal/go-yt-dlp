# go-yt-dlp

Servidor Go + Web UI para descargar contenido usando `yt-dlp`.

## Requisitos
- Go 1.26+
- `yt-dlp` disponible (PATH o raiz del proyecto)
- `ffmpeg` y `ffprobe` (default `C:\Shared\ffmpeg\bin`)

## Inicio rapido
1. Copia `.env.example` a `.env`.
2. Ajusta variables necesarias.
3. Ejecuta:

```bash
go run ./cmd/server
```

## Cookies de YouTube (manual)
Cuando YouTube pide login anti-bot, usa un `cookies.txt` en formato Netscape.

### Opcion A (recomendada): exportar desde extension
1. Instala la extension de navegador `Get cookies.txt LOCALLY`.
2. Abri `youtube.com` con la sesion logueada.
3. Exporta cookies a archivo `cookies.txt`.
4. Guarda el archivo en la raiz del proyecto.

### Opcion B: exportar con herramienta CLI
- Usa una herramienta que exporte cookies en formato Netscape.
- Guarda el resultado como `cookies.txt` en la raiz del proyecto.

### Configuracion .env
Usa una sola estrategia:

```env
# Estrategia 1: archivo manual
YTDLP_COOKIES_FILE=.\cookies.txt
YTDLP_COOKIES_FROM_BROWSER=

# Estrategia 2: browser directo (si funciona en tu equipo)
YTDLP_COOKIES_FROM_BROWSER=chrome
YTDLP_COOKIES_FILE=
```

### Validaciones implementadas
En el arranque, el servidor valida `YTDLP_COOKIES_FILE`:
- el archivo debe existir
- no puede ser un directorio
- debe tener header y filas validas de formato Netscape

Si no pasa, el servidor no inicia.

## Variables de entorno relevantes
- `LISTEN_ADDR` (default `:8080`)
- `YTDLP_BIN` (default `yt-dlp`)
- `YTDLP_JS_RUNTIMES` (default `deno,node`)
- `YTDLP_COOKIES_FILE`
- `YTDLP_COOKIES_FROM_BROWSER`
- `YTDLP_AUTO_UNLOCK_BROWSER_COOKIES` (default `true` en Windows)
- `FFMPEG_LOCATION` (default `C:\Shared\ffmpeg\bin`)

## Seguridad basica
- No subas `cookies.txt` al repositorio.
- Regenera cookies cuando expiren.

