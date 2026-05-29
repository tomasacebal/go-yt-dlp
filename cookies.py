import os
import time
import schedule
from playwright.sync_api import sync_playwright

# --- CONFIGURACIÓN ---
OUTPUT_DIR = r"C:\Shared\go-yt-dlp"
COOKIE_FILE_PATH = os.path.join(OUTPUT_DIR, "cookies.txt")

# Carpeta donde Playwright guardará tu sesión, historial y cookies.
USER_DATA_DIR = os.path.join(OUTPUT_DIR, "playwright_profile")

# ¡IMPORTANTE! 
# 1. Pon esto en True la primera vez que lo ejecutes.
# 2. Inicia sesión manualmente en la ventana que se abre.
# 3. Cierra la ventana.
# 4. Pon esto en False y vuelve a ejecutar el script para dejarlo en segundo plano.
INITIAL_LOGIN_MODE = False  

def export_netscape_format(cookies, filepath):
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write("# Netscape HTTP Cookie File\n")
        f.write("# Este archivo fue generado automaticamente.\n\n")
        
        for cookie in cookies:
            domain = cookie.get('domain', '')
            include_subdomains = "TRUE" if domain.startswith('.') else "FALSE"
            path = cookie.get('path', '/')
            secure = "TRUE" if cookie.get('secure', False) else "FALSE"
            
            expires = cookie.get('expires', -1)
            expires_int = int(expires) if expires > 0 else 0
            
            name = cookie.get('name', '')
            value = cookie.get('value', '')
            
            f.write(f"{domain}\t{include_subdomains}\t{path}\t{secure}\t{expires_int}\t{name}\t{value}\n")

def fetch_youtube_cookies():
    print(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] Iniciando extracción de cookies (Perfil Persistente)...")
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    with sync_playwright() as p:
        # Aquí está la clave: usamos launch_persistent_context en lugar de launch
        # Usamos INITIAL_LOGIN_MODE para determinar si mostramos la ventana o no
        context = p.chromium.launch_persistent_context(
            user_data_dir=USER_DATA_DIR,
            headless=not INITIAL_LOGIN_MODE, 
            channel="chrome" # A veces Google bloquea inicios de sesión en Chromium genérico, forzamos usar Chrome.
        )
        
        page = context.pages[0] if context.pages else context.new_page()
        
        try:
            page.goto("https://www.youtube.com/")
            
            if INITIAL_LOGIN_MODE:
                print("Por favor, inicia sesión en la ventana del navegador.")
                print("El script esperará 5 minutos, o hasta que cierres la ventana.")
                # Le damos mucho tiempo para que resuelvas el 2FA o reCAPTCHAs
                page.wait_for_timeout(300000) 
            else:
                # Si estamos de fondo, esperamos 10 segundos para que todo cargue con tu sesión
                page.wait_for_load_state("networkidle")
                page.wait_for_timeout(10000)
            
            cookies = context.cookies()
            export_netscape_format(cookies, COOKIE_FILE_PATH)
            
            print(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] Exito: {len(cookies)} cookies exportadas a {COOKIE_FILE_PATH}")
            
        except Exception as e:
            print(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] Error: {e}")
        finally:
            context.close()

def main():
    fetch_youtube_cookies()
    
    if INITIAL_LOGIN_MODE:
        print("Modo de inicio de sesión finalizado. Cambia INITIAL_LOGIN_MODE a False en el código y reinicia.")
        return

    # Programar la ejecución en segundo plano
    schedule.every(8).hours.do(fetch_youtube_cookies)
    print("Script en ejecución de fondo. Presiona Ctrl+C para detener el proceso.")
    
    while True:
        schedule.run_pending()
        time.sleep(60)

if __name__ == "__main__":
    main()