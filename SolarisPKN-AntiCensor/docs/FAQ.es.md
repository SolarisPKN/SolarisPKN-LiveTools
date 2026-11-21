# ❓ Preguntas frecuentes — AntiCensor
[English](FAQ.en.md) · [Volver](../README.es.md)

### 🎮 ¿Roblox recibe mi voz real?
**No a través de AntiCensor.** La aplicación genera Lelele sintético. **Otras aplicaciones** que capturen tu micrófono físico sí pueden recibirlo si así las configurás. Seleccioná CABLE Output/SolarisPKN-Mic Lelele en Roblox.

### 🎙️ ¿Por qué CABLE Input es una salida?
VB-Audio usa el nombre desde la perspectiva del cable: **CABLE Input** recibe reproducción de AntiCensor. **CABLE Output** entrega la captura virtual al juego.

### 🧠 ¿Reconoce palabras?
No. Usa **WebRTC VAD clásico** para estimar actividad vocal y RMS para el volumen. No identifica frases, no transcribe ni necesita un modelo de lenguaje.

### 🚚 ¿Elimina clics, camiones o música?
Reduce algunos falsos disparos. No es una cancelación de ruido profesional ni AEC. Un camión fuerte, la música y voces de juegos pueden producir falsos positivos. Primero corregí fuentes mezcladas.

### 🔁 ¿Qué hace Modo experto?
Muestra nombres VB-CABLE y diagnóstico **en AntiCensor**. No los deshabilita en Windows. Si no necesitás verlos, dejalo apagado.

### 🎧 ¿Por qué la barra Lelele sigue moviéndose con música?
Podría ser que el Gadnic reciba el juego y active la síntesis; otra aplicación también puede reproducir directamente en CABLE Input. Detener AntiCensor con música sonando ayuda a distinguir ambas causas. En OBS, evitar monitorización Lelele a auriculares si no es necesaria.

### 📺 ¿Puedo usar mi voz en OBS y Lelele en Roblox?
Sí, configurando OBS con el micrófono físico y el juego con el virtual. **Siempre verificá qué fuente envía cada app**.

### 🔐 ¿Es una herramienta para saltar restricciones de voz?
No. Usala respetando las reglas de cada juego y sus requisitos de chat de voz.

### 🧩 ¿Es gratis?
El código de AntiCensor es GPLv3. VB-CABLE básico es donationware independiente y su licencia pertenece al fabricante.

💚 [SolarisPKN-Labs](https://labs.solarispkn.com.ar/)