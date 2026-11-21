# Instalar SolarisPKN-AntiCensor v0.8.0

[English](README_INSTALADOR.md) · [README del módulo](README.es.md)

Windows 10/11 x64. El instalador **SolarisPKN-AntiCensor-Setup-v0.8.0.exe** incluye la aplicación GPLv3 con **modo oscuro Solaris Tray**, menú bilingüe y solo LELELE.

**VB-CABLE**, del fabricante VB-Audio Software, es una dependencia externa propietaria donationware. Si falta, el instalador permite instalarlo con su herramienta oficial (UAC y reinicio). Si existe, **no ofrece reinstalarlo**: muestra únicamente **Limpiar driver (solo errores)** con advertencia y confirmación; abre la utilidad del fabricante y no borra nada automáticamente. No desactiva Secure Boot.

Al instalar o iniciar, AntiCensor intenta poner al dispositivo de grabación el nombre **SolarisPKN-Mic Lelele**, aunque algunos juegos podrían mostrar **CABLE Output**.

**Calibración manual:** silencio 5 s → susurro 5 s → voz normal 5 s → grito opcional 5 s. Cada etapa requiere un clic explícito; podés repetir, volver y cancelar. Sin grito se estima el máximo.

**Ajustes de updates:** checkbox semanal opcional; desmarcado, solamente chequeo manual. Los paquetes oficiales se descargan, comprueban con SHA-256 y **se instalan solo al cerrar AntiCensor o al próximo inicio de Windows**, nunca durante el stream. Fuente fija: **SolarisPKN/SolarisPKN-LiveTools/SolarisPKN-AntiCensor**.

Desinstalar AntiCensor desde Aplicaciones no desinstala VB-CABLE. Condiciones de VB-Audio: https://vb-audio.com/Services/licensing.htm. Pendientes pruebas reales con Roblox/OBS.
