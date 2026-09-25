# Sistema de alertas — CPD Monitor

Documentación de referencia de todo lo configurado en Grafana Alerting.
Última revisión: 23/09/2026.

---

## 1. Resumen

Cuatro reglas, cada una con un propósito único, todas provisionadas por
fichero (`grafana/provisioning/alerting/rules.yaml`) y evaluadas cada 30s
para un aviso lo más rápido posible sin caer en ruido por picos de un
solo ciclo:

| Regla | Grupo de evaluación | Dispara cuando | Frecuencia de aviso |
|---|---|---|---|
| `Temperatura CPD fuera de rango` | `CPD-Ambiente` (30s) | Valor real fuera de 18-33°C, sostenido 30s | Cada 30 min mientras persista |
| `Humedad CPD fuera de rango` | `CPD-Ambiente` (30s) | Valor real fuera de 25-60%, sostenido 30s | Cada 30 min mientras persista |
| `Batería CPD baja` | `CPD-Bateria-Baja` (30s) | Batería de algún sensor <10%, sostenido 30s | Cada 30 min mientras persista |
| `Sensor CPD sin datos` | `CPD-Sin-Datos` (30s) | Más de 10 min desde la última lectura de un sensor concreto | Un único aviso hasta que se recupera |

---

## 2. Reglas de alerta (Alerting → Alert rules)

Carpeta común: `CPD - Monitorización ambiental`. Las 4 reglas tienen
`provenance: file` — se editan en `grafana/provisioning/alerting/rules.yaml`,
nunca desde la interfaz (que lo bloquea con un aviso de "provisioned").

### 2.1 — `Temperatura CPD fuera de rango`

- **Query (Flux):** último valor de `temperatura_c` en `cpd_ambiente`.
- **Reduce (B):** `Last`, modo **`Drop Non-Numeric Value`**.
- **Threshold (C):** `IS OUTSIDE RANGE 18 – 33`.
- **Grupo de evaluación:** `CPD-Ambiente`, intervalo `30s`.
- **Periodo pendiente:** `30s`.
- **Estado si no hay datos:** `No Data` (no alerta — ese caso lo cubre
  `Sensor CPD sin datos` en exclusiva).
- **Estado si hay error de consulta:** `Error` (no alerta — ver 5.5, es
  la causa del falso `-1.0` que se coló incluso después de arreglar 5.1).
- **Faltan evaluaciones de series para resolver:** `10` (ver 5.2).

### 2.2 — `Humedad CPD fuera de rango`

Igual que la anterior, sobre `humedad_rel_pct`, rango `25 – 60`, mismos
ajustes de grupo/pendiente/evaluaciones-para-resolver/execErrState.

### 2.3 — `Batería CPD baja`

- **Query (Flux):**
  ```flux
  from(bucket: "cpd_monitorizacion")
    |> range(start: -15m)
    |> filter(fn: (r) => r._measurement == "cpd_ambiente")
    |> filter(fn: (r) => r._field == "bateria_pct")
    |> last()
    |> keep(columns: ["_time", "_value", "sensor_id", "ubicacion"])
  ```
- **Reduce (B):** `Last`, modo `Drop Non-Numeric Value` (añadido el
  23/09/2026 por consistencia con las otras reglas — antes no lo tenía).
- **Threshold (C):** `IS BELOW 10`.
- **Grupo de evaluación:** `CPD-Bateria-Baja`, intervalo `30s`.
- **Estado si no hay datos / error:** `No Data` / `Error` (ninguno alerta).
- **Etiqueta:** `tipo=bateria_baja`.

### 2.4 — `Sensor CPD sin datos`

Rediseñada por completo el 22-23/09/2026 — ver 5.6 para el porqué.

- **Query (Flux):**
  ```flux
  from(bucket: "cpd_monitorizacion")
    |> range(start: -30d)
    |> filter(fn: (r) => r._measurement == "cpd_ambiente")
    |> filter(fn: (r) => r._field == "temperatura_c")
    |> last()
    |> map(fn: (r) => ({ r with _value: float(v: int(v: now()) - int(v: r._time)) / 1000000000.0 / 60.0 }))
    |> keep(columns: ["_time", "_value", "sensor_id", "ubicacion"])
  ```
  En vez de contar lecturas en una ventana corta, calcula **los minutos
  transcurridos desde la última lectura de cada sensor**, sobre una
  ventana de 30 días. Así, cualquier sensor que alguna vez haya
  reportado sigue devolviendo una fila (con un contador que crece si deja
  de emitir) — cubre automáticamente cualquier sensor presente o futuro,
  sin necesidad de escribir su `id` en ningún sitio de esta regla.
- **Reduce (B):** `Last`. **Threshold (C):** `IS ABOVE 10` (minutos).
- **Grupo de evaluación:** `CPD-Sin-Datos`, intervalo `30s`.
- **Periodo pendiente:** `30s`.
- **Estado si no hay datos:** `Alerting` (aquí sí — si ni siquiera hay
  fila para ningún sensor en 30 días, algo va muy mal).
- **`notification_settings`:** `receiver: email-cpd`, `group_wait: 10s`,
  `group_interval: 10s`, `repeat_interval: 1d` — fijado a nivel de la
  propia regla, no de la política (ver 5.3, la política resultó frágil
  para esto).
- **Etiqueta:** `tipo=sin_datos`.
- **Se auto-resuelve** en cuanto vuelve a haber una lectura reciente del
  sensor, sin ninguna acción manual — confirmado en vivo el 22/09/2026
  con el propio sensor 1.

---

## 3. Grupos de evaluación (Alerting → Alert rules → Grupos)

| Grupo | Intervalo | Reglas |
|---|---|---|
| `CPD-Ambiente` | 30s | Temperatura, Humedad |
| `CPD-Bateria-Baja` | 30s | Batería CPD baja |
| `CPD-Sin-Datos` | 30s | Sensor CPD sin datos |

Los 3 grupos comparten intervalo de 30s desde el 23/09/2026 (antes,
batería estaba a 5m — hasta 10 min de espera para un aviso). El grupo
genérico original (`Monitorizacion-CPD`, 10s) quedó sin uso tras la
reorganización del 16/09/2026.

---

## 4. Política de notificaciones (Alerting → Notification policies)

Fichero: `grafana/provisioning/alerting/policies.yaml`.

- **Política raíz:** contacto `email-cpd`, `group_by: ['alertname', 'sensor_id']`,
  `repeat_interval: 30m`.
- **Ruta hija:** `Sensor CPD sin datos` (matcher `=~`), mismo `group_by`,
  `repeat_interval: 1d` (aunque el `repeat_interval` real de esa regla lo
  fija su propio `notification_settings`, ver 2.4).

**Limitación conocida — agrupación por `sensor_id` no siempre separa el
email.** Pese al `group_by` correcto tanto en la política como a nivel de
regla, cuando **dos sensores** disparan la misma alerta en la misma
ventana de `group_wait`, llega **un solo email con las dos instancias**
en vez de dos emails separados. Investigado a fondo el 23/09/2026 (ver
5.7): causa raíz encontrada y confirmada, sin solución disponible con
provisioning por fichero en Grafana 13.0.2. Comportamiento aceptado como
correcto para producción: si solo un sensor tiene el problema (el caso
habitual), llega su propio email individual sin mezclarse con nada.

---

## 5. Historial de problemas resueltos

### 5.1 — Falso `-1.0°C` / `0.0°C`, primera vuelta (14/09/2026)

**Síntoma:** con el sensor desconectado, llegaban alertas de fuera de
rango con valores inventados, no reales.

**Arreglo:** modo `Reduce` cambiado de `Replace Non-Numeric Value` a
`Drop Non-Numeric Value`; `No Data` de temperatura/humedad cambiado a
no-alerting; alerta interna `DatasourceNoData` de Grafana silenciada.

### 5.2 — Resolución falsa por "series obsoleta" (16/09/2026)

**Síntoma:** con la humedad real estable y fuera de rango, la alerta se
resolvía sola cada pocos minutos sin que el valor real cambiara.

**Causa:** "Faltan evaluaciones de series para resolver" en su valor por
defecto (`2`) combinado con el grupo de evaluación a 10s — cualquier
hueco de 2 ciclos sin dato (un fallo BLE puntual) resolvía la alerta sola.

**Arreglo:** grupo de evaluación separado a 1 minuto + evaluaciones para
resolver subido a `10`.

### 5.3 — Repetición cada 30 min pese al `repeat_interval` de 24h (15-16/09/2026)

**Síntoma:** `Sensor CPD sin datos` seguía enviando un email cada ~30 min
pese a una ruta de política con `repeat_interval: 1d` y matcher correcto.

**Causa real:** el matcher por `alertname` con operador exacto (`=`) no
incluía el emoji/sufijo completo del nombre de la regla, así que nunca
hacía match y la notificación caía en la política raíz (30m). Cambiarlo a
`=~` (contiene) tampoco lo arregló de forma fiable.

**Arreglo definitivo:** `repeat_interval` fijado **directamente en la
regla** (`notification_settings`), sin depender de ningún matcher de
política. Sigue así hoy (ver 2.4).

### 5.4 — Colector colgado sin loguear (14-16/09/2026)

**Síntoma:** `systemctl status` decía `active (running)` pero llevaba casi
2 días sin escribir ni una línea de log.

**Causa exacta:** no confirmada del todo (posible llamada BLE/D-Bus que
nunca devolvió ni error ni resultado).

**Arreglo aplicado:** reinicio manual del servicio.

**Mitigación parcial añadida el 23/09/2026:** el colector ahora escribe
un evento en InfluxDB (`cpd_eventos`) en cada transición de
conexión/desconexión real, visible en el dashboard — no detecta un
cuelgue "vivo pero sin avanzar" (para eso hace falta el watchdog de
systemd, sección 8), pero si el colector se cuelga también deja de
escribir datos, así que `Sensor CPD sin datos` lo detecta de todas formas
en menos de 11 minutos.

### 5.5 — Falso `-1.0°C`, segunda vuelta (22/09/2026)

**Síntoma:** meses después de 5.1, seguía llegando ocasionalmente un
`-1.0°C`/`-1.0%H` puntual, pese a tener ya `Drop Non-Numeric Value` y
`No Data` en no-alerting.

**Causa real, distinta a la de 5.1:** `execErrState` (estado ante un
**error** de consulta, no ante ausencia de datos) seguía en `Alerting` en
las reglas de temperatura/humedad. Un fallo puntual de InfluxDB (por
ejemplo, durante un `docker compose up -d` de despliegue) hacía que la
expresión de reduce nunca llegara a calcularse, y el email formateaba un
valor sin inicializar como `-1.0`.

**Arreglo:** `execErrState` cambiado a `Error` (no alerta) en las dos
reglas — la detección de "sin datos/colector caído" la cubre en exclusiva
la regla dedicada, sin este efecto secundario. Ver `CHANGELOG.md`
22/09/2026.

### 5.6 — Watchdog de "sin datos" ciego a un sensor concreto (22/09/2026)

**Síntoma:** el sensor 1 llevó más de 2 horas sin emitir por Bluetooth
(confirmado en `journalctl`, error repetido cada minuto) sin que llegara
ninguna alerta de "sin datos".

**Causa real:** la consulta original contaba lecturas de **todos los
sensores juntos** (`count()` sin filtrar). Como el sensor 2 seguía
reportando, la consulta nunca devolvía una tabla vacía — Flux simplemente
omite del resultado cualquier serie sin puntos en la ventana, en vez de
devolver un `0` explícito para ella. La regla nunca llegaba a evaluar
"sin datos" para el sensor 1 mientras el sensor 2 siguiera vivo.

**Solución evaluada y descartada:** una regla por sensor (con script de
mantenimiento para crear/borrar una por cada sensor nuevo) — funcionaba,
pero no escalaba sin tocar ficheros al añadir un sensor.

**Arreglo definitivo:** regla única basada en tiempo transcurrido desde
la última lectura (ver 2.4) — cubre cualquier sensor automáticamente, sin
mantenimiento.

### 5.7 — Agrupación por `sensor_id` no separa el email cuando coinciden dos sensores (23/09/2026)

**Síntoma:** con `group_by: ['alertname', 'sensor_id']` en la política, y
también probado a nivel de regla (`notification_settings.group_by`), dos
sensores disparando la misma alerta a la vez seguían llegando en un solo
email en vez de dos.

**Investigación, en orden:**
1. Agrupar por `sensor_id` combinado con `alertname` en la política → sin
   efecto.
2. Agrupar solo por `sensor_id` en la política → sin efecto (descarta que
   fuera un problema de combinar dos etiquetas).
3. Rutas de política separadas por `sensor_id` (`object_matchers`
   distintos por sensor) → sin efecto.
4. Prueba con una alerta **nunca antes disparada** (huella/fingerprint
   100% nueva, para descartar "memoria pegada" de Alertmanager entre
   pruebas anteriores) → sin efecto, confirma que no era un problema de
   estado residual.
5. `group_by` a nivel de la propia regla (`notification_settings.group_by`,
   la opción "Anular agrupación" de la interfaz) → sin efecto tampoco.

**Causa raíz encontrada** (consultando la configuración *viva* de
Alertmanager vía `GET /api/alertmanager/grafana/config/api/v1/alerts`,
no la que devuelve la API de "provisioning"): Grafana inyecta
automáticamente, por delante de cualquier ruta propia, una rama interna
que coincide con el 100% de las alertas
(`__grafana_autogenerated__=true`), y dentro de ella **fuerza
`group_by: ["grafana_folder", "alertname"]`**, ignorando cualquier
`group_by` puesto por encima vía provisioning de fichero. Como
Alertmanager evalúa rutas en orden y se detiene en la primera coincidencia,
esta rama interna intercepta la alerta antes de que llegue a evaluarse
nuestra configuración. La opción "Anular agrupación" de la interfaz
gráfica (que si funciona, según otro proyecto de referencia) usa
aparentemente una vía distinta (`alertingPolicyRoutingSettings`, una API
más nueva) no expuesta todavía en el formato de fichero que provisionamos
aquí.

**Estado:** aceptado como limitación conocida de esta versión/config, no
como bug propio. Sin impacto funcional real: cuando solo un sensor tiene
el problema (el caso habitual), su email llega individual sin mezclarse
con nada; solo se agrupan cuando **coinciden en el tiempo** dos sensores
con la misma alerta, lo cual además ahorra ruido de bandeja de entrada.

---

## 6. Silencios (Alerting → Silences)

- **`alertname = DatasourceNoData`**, duración ~5 años (14/09/2026 →
  13/09/2031). Evita el aviso interno y redundante de Grafana cuando el
  datasource no devuelve datos; las reglas propias ya cubren ese caso de
  forma explícita. Menos crítico desde que `noDataState`/`execErrState`
  están bien puestos en las 4 reglas (5.5), pero se deja como red de
  seguridad adicional.

> Este silencio vive en la base de datos de Grafana, no está provisionado
> por fichero — a diferencia de las 4 reglas, que sí lo están desde el
> 22/09/2026.

---

## 7. Contact point y plantilla de email

Fichero: `grafana/provisioning/alerting/contactpoints.yaml`.

La plantilla distingue **cuatro** casos al construir el cuerpo del email
(detectando por `contains "sin datos" .Labels.alertname`, no por
coincidencia exacta — más robusto ante cambios de texto del título):

1. **`Sensor CPD sin datos`** → mensaje dedicado con `sensor_id`/`ubicacion`,
   sin intentar mostrar ningún valor numérico.
2. **`Batería CPD baja`** → muestra el nivel en `%` (referencia `A`), o un
   texto de "no se ha podido leer" si no hay valor disponible.
3. **Temperatura/Humedad, con valor real disponible** → formato clásico
   (`Valor: X.X°C (rango normal: 18-33°C)` / `X.X%H (rango normal: 25-60%H)`).
4. **Temperatura/Humedad, sin valor disponible** (`len .Values == 0`) →
   "No se han recibido lecturas de este sensor".

El enlace al dashboard usa `${GRAFANA_PUBLIC_URL}` (variable de `.env`);
los botones propios de Grafana (Silence/View dashboard/View panel) usan
`GF_SERVER_ROOT_URL` en `docker-compose.yml` — sin este segundo, esos
botones apuntan a `localhost` en vez de a la IP real, aunque el texto del
email esté bien.

---

## 8. Pendiente

Ver `docs/MEJORAS-FUTURAS.md` para el detalle completo:
- Watchdog de systemd para el colector (sección 5.4) — sigue siendo la
  única mejora de alta prioridad realmente pendiente de esta lista.
- Vigilar si la limitación de agrupación (5.7) cambia con una futura
  actualización de Grafana.
