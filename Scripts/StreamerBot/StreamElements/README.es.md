# StreamElements Points Fetcher for Streamer.bot

Obtiene los puntos de un usuario desde StreamElements y los almacena en una variable de Streamer.bot para utilizarla en comandos, acciones, respuestas automáticas y automatizaciones.

## Características

* Consulta puntos de usuarios mediante la API de StreamElements.
* Compatible con Streamer.bot.
* Guarda el resultado en la variable `%sepoints%`.
* Detecta automáticamente las variables:

  * `username`
  * `user`
  * `targetUser`
* Manejo de errores de red y API.
* Registro detallado en los logs de Streamer.bot.

---

## Requisitos

* Streamer.bot
* Cuenta de StreamElements
* Token JWT de StreamElements
* Channel ID de StreamElements

---

## Configuración

Dentro del script reemplaza:

```csharp
private readonly string _jwtToken = "YOUR_JWT_TOKEN";
private readonly string _channelId = "YOUR_CHANNEL_ID";
```

### Obtener el JWT Token

1. Inicia sesión en StreamElements.
2. Ve a Account Settings.
3. Genera o copia tu JWT Token.

### Obtener el Channel ID

Puedes encontrarlo en la URL de tu dashboard de StreamElements o utilizando la API.

---

## Uso

Ejecuta la acción desde Streamer.bot proporcionando cualquiera de las siguientes variables:

* `username`
* `user`
* `targetUser`

El script realizará la consulta y almacenará el resultado en:

```text
%sepoints%
```

### Ejemplo

Si el usuario tiene 1250 puntos:

```text
%sepoints% = 1250
```

Luego podrás usarlo en respuestas:

```text
@%user% tiene %sepoints% puntos.
```

---

## Variables generadas

| Variable     | Descripción                                      |
| ------------ | ------------------------------------------------ |
| `%sepoints%` | Cantidad de puntos del usuario en StreamElements |

---

## Funcionamiento

1. Obtiene el nombre del usuario.
2. Realiza una consulta a la API de StreamElements.
3. Procesa la respuesta JSON.
4. Extrae el valor de puntos.
5. Guarda el resultado en `%sepoints%`.
6. Registra información y errores en el log.

---

## Manejo de errores

Si ocurre algún problema:

* Usuario no encontrado.
* Token inválido.
* Error de conexión.
* Respuesta inesperada de la API.

El script devolverá:

```text
0
```

y registrará el motivo en los logs de Streamer.bot.

---

## Licencia

Este proyecto se distribuye bajo la licencia MIT.
