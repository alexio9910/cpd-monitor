# Sistema de alertas — CPD Monitor

Documentación de referencia de todo lo configurado en Grafana Alerting.
Última revisión: 16/09/2026.

---

## 1. Resumen

Cuatro reglas, cada una con un propósito único y su propio grupo de
evaluación, para que cada situación dispare exactamente lo que corresponde
y nada más:

| Regla | Grupo de evaluación | Dispara cuando | Frecuencia de aviso |
|---|---|---|---|
| `Temperatura CPD fuera de rango` | `CPD-Ambiente` (1m) | Valor real fuera de 18-27°C | Cada 30 min mientras persista |
| `Humedad CPD fuera de rango` | `CPD-Ambiente` (1m) | Valor real fuera de 40-60% | Cada 30 min mientras persista |
| `Sensor CPD sin datos` | `CPD-Sin-Datos` (30s) | Sin ninguna lectura en 10 min | Un único aviso hasta que se recupera |
| `Batería CPD baja` | `CPD-Bateria-Baja` (5m) | Batería de algún sensor <10% | Cada 30 min mientras persista |

---

## 2. Reglas de alerta (Alerting → Alert rules)

Carpeta común: `CPD - Monitorización ambiental`.

### 2.1 — `Temperatura CPD fuera de rango`

- **Query (Flux):** último valor de `temperatura_c` en `cpd_ambiente`.
- **Reduce (B):** `Last`, modo **`Drop Non-Numeric Value`**.
- **Threshold (C):** `IS OUTSIDE RANGE 18 – 27`.
- **Grupo de evaluación:** `CPD-Ambiente`, intervalo `1m`.
- **Periodo pendiente:** `1m`.
- **Estado si no hay datos:** `No Data` (no `Alerting` — ese caso lo cubre
  `Sensor CPD sin datos` en exclusiva).
- **Faltan evaluaciones de series para resolver:** `10` (ver sección 5.2 —
  el valor por defecto, `2`, causaba resoluciones falsas).

### 2.2 — `Humedad CPD fuera de rango`

Igual que la anterior, sobre `humedad_rel_pct`, rango `40 – 60`, mismos
ajustes de grupo/pendiente/evaluaciones-para-resolver.

### 2.3 — `Sensor CPD sin datos`

- **Query (Flux):**
  ```flux
  from(bucket: "cpd_monitorizacion")
    |> range(start: -10m)
    |> filter(fn: (r) => r._measurement == "cpd_ambiente")
    |> filter(fn: (r) => r._field == "temperatura_c")
    |> count()
  ```
- **Reduce:** `Last`. **Threshold:** `IS BELOW 1`.
- **Grupo de evaluación:** `CPD-Sin-Datos`, intervalo `30s`.
- **Periodo pendiente:** `2m`.
- **Estado si no hay datos:** `Alerting` (aquí sí — si la query devuelve
  tabla vacía, también cuenta como incidencia).
- **Silencio, agrupación y temporización → Anular tiempos (a nivel de
  regla):** `Group wait 10s / Group interval 10s / Repeat interval 24h`.
  Un único aviso por incidencia en vez de cada 30 min. Este ajuste vive
  **en la propia regla**, no en `policies.yaml` — ver sección 5.3 para el
  porqué (se intentó primero por política de notificación y no funcionó
  de forma fiable).
- **Etiqueta:** `tipo=sin_datos`.

### 2.4 — `Batería CPD baja`

- **Query (Flux):**
  ```flux
  from(bucket: "cpd_monitorizacion")
    |> range(start: -15m)
    |> filter(fn: (r) => r._measurement == "cpd_ambiente")
    |> filter(fn: (r) => r._field == "bateria_pct")
    |> last()
    |> keep(columns: ["_time", "_value", "sensor_id", "ubicacion"])
  ```
- **Threshold:** `IS BELOW 10`.
- **Grupo de evaluación:** `CPD-Bateria-Baja`, intervalo `5m`.
- **Periodo pendiente:** `5m`.
- **Estado si no hay datos:** `No Data`.
- **Etiqueta:** `tipo=bateria_baja`.
- **Referencia de valor en la plantilla de email:** `A` (esta regla no
  tiene un bloque Reduce/Threshold separado como B/C, la condición se
  define directamente sobre la consulta `A`).

> ⚠️ **Las 4 reglas se crearon desde la interfaz de Grafana y viven en su
> base de datos interna (SQLite), no están provisionadas por fichero.**
> Si se hace `docker compose down -v` (borrado de volúmenes), hay que
> recrearlas a mano siguiendo esta documentación. Ver tarea pendiente en
> `docs/MEJORAS-FUTURAS.md`.

---

## 3. Grupos de evaluación (Alerting → Alert rules → Grupos)

| Grupo | Intervalo | Reglas |
|---|---|---|
| `CPD-Ambiente` | 1m | Temperatura, Humedad |
| `CPD-Sin-Datos` | 30s | Sensor CPD sin datos |
| `CPD-Bateria-Baja` | 5m | Batería CPD baja |

El grupo genérico original (`Monitorizacion-CPD`, 10s) quedó sin uso tras
la reorganización del 16/09/2026 — evaluaba con demasiada frecuencia para
lo que necesita cada tipo de dato, y fue la causa raíz de dos bugs (ver
sección 5).

---

## 4. Política de notificaciones (Alerting → Notification policies)

Fichero: `grafana/provisioning/alerting/policies.yaml`.

- **Política raíz:** contacto `email-cpd`, `repeat_interval: 30m`. Aplica
  a `Temperatura`, `Humedad` y `Batería CPD baja`.
- No hay rutas anidadas — el aviso único de `Sensor CPD sin datos` se
  resuelve con "Anular tiempos" a nivel de esa regla (ver 2.3 y 5.3), no
  con una ruta de política.

---

## 5. Historial de problemas resueltos

### 5.1 — Falso `-1.0°C` / `0.0°C` (14/09/2026)

**Síntoma:** con el sensor desconectado, llegaban alertas de fuera de
rango con valores inventados, no reales (confirmado consultando InfluxDB
directamente: no había ningún punto con ese valor).

**Causas combinadas y arreglo:**
1. El modo `Reduce` estaba en `Replace Non-Numeric Value` → cambiado a
   `Drop Non-Numeric Value`.
2. El estado `No Data` de las reglas de temperatura/humedad estaba en
   `Alerting`, y la plantilla del email formateaba un valor inexistente
   como `0.0` → separado en la regla dedicada `Sensor CPD sin datos`
   (sección 2.3), y `No Data` cambiado a no-alerting en temperatura/humedad.
3. La alerta interna `DatasourceNoData` de Grafana generaba un aviso
   redundante → silenciada (Alerting → Silences, matcher
   `alertname=DatasourceNoData`, ~5 años de duración).

### 5.2 — Resolución falsa por "series obsoleta" (16/09/2026)

**Síntoma:** con la humedad real estable y establemente fuera de rango
(ej. 36-37%, confirmado en los logs del colector), la alerta se resolvía
sola cada pocos minutos y se volvía a disparar, generando pares
`RESUELTA`/`ALERTA` sin que el valor real hubiera cambiado.

**Causa:** el campo **"Faltan evaluaciones de series para resolver"**
tenía su valor por defecto (`2`). Con el grupo de evaluación en 10s (el
genérico original), cualquier hueco puntual de 2 evaluaciones seguidas sin
dato (por ejemplo, un ciclo BLE fallido del colector) hacía que Grafana
diera la serie por "perdida" y resolviera la alerta automáticamente,
aunque el último valor real conocido siguiera fuera de rango.

**Arreglo:** grupo de evaluación separado a 1 minuto (`CPD-Ambiente`) +
"Faltan evaluaciones de series para resolver" subido a `10` en ambas
reglas — da 10 minutos de margen real antes de considerar la serie
perdida, tiempo de sobra para que el colector se recupere de un fallo BLE
puntual sin que la alerta parpadee.

### 5.3 — Repetición cada 30 min pese al `repeat_interval` de 24h (15-16/09/2026)

**Síntoma:** `Sensor CPD sin datos` seguía enviando un email cada ~30 min
en vez de uno solo, pese a que la API de Grafana confirmaba una ruta de
notificación con `repeat_interval: 1d` y el matcher correcto.

**Intentos fallidos, en orden:**
1. Ruta por etiqueta `tipo=sin_datos` con operador `=` → sin efecto
   aparente (causa nunca confirmada del todo).
2. Ruta por `alertname` con operador `=` (coincidencia exacta) → **causa
   real encontrada**: el valor puesto en el matcher no incluía el emoji ni
   el sufijo completo del nombre de la regla, así que nunca hacía match
   contra la etiqueta real (`🔴 Sensor CPD sin datos - Posible fallo de
   monitorización`) y la notificación caía en la política raíz (30m).
3. Ruta por `alertname` con operador `=~` (contiene) y fragmento parcial
   `Sensor CPD sin datos` → seguía repitiendo cada 30 min pese a que la
   API mostraba la configuración correcta.

**Arreglo definitivo:** en vez de depender del enrutamiento por política
(frágil ante el emoji/nombre exacto y con comportamiento inconsistente en
esta versión), se configuró el `repeat_interval` **directamente en la
regla**, con el toggle **"Anular tiempos"** en la sección "5. Configure
notifications" de la propia regla `Sensor CPD sin datos` — sin depender de
ningún matcher externo. Ver 2.3.

### 5.4 — Colector colgado sin loguear (14-16/09/2026)

**Síntoma:** el servicio `cpd-monitor` aparecía como `active (running)` en
`systemctl status`, pero llevaba casi 2 días sin escribir ni una sola
línea de log (ni éxito ni error), cuando debería registrar un intento cada
60s sin parar.

**Causa exacta:** no confirmada del todo — el proceso quedó bloqueado de
alguna forma (posiblemente una llamada BLE/D-Bus que nunca devolvió ni
error ni resultado) sin que systemd lo detectara como fallido, porque el
proceso seguía "vivo" a nivel de sistema operativo aunque no avanzara.

**Cómo se detectó:** por casualidad, al comparar el dashboard ("Sin
datos") contra un email de "RESUELTA" que no cuadraba con la realidad.

**Arreglo aplicado:** `sudo systemctl restart cpd-monitor` — el reinicio
manual lo destrabó (`Killing process ... with signal SIGKILL`,
`Failed with result 'timeout'` en los logs de systemd).

**Riesgo pendiente:** este tipo de fallo (colgado pero "activo") no está
cubierto por la regla `Sensor CPD sin datos` de forma directa, porque esa
regla se basa en si llegan datos a InfluxDB, no en si el proceso del
colector responde — de hecho, si el colector se cuelga, SÍ deja de
escribir datos, así que la regla sí lo habría detectado tarde o temprano
igualmente. Pero un watchdog a nivel de systemd (`WatchdogSec` +
notificación `sd_notify` desde el propio colector) detectaría el cuelgue
mucho antes y lo reiniciaría solo, sin depender de que pasen los 10 min de
la regla de "sin datos". Ver tarea nueva en `docs/MEJORAS-FUTURAS.md`.

---

## 6. Silencios (Alerting → Silences)

- **`alertname = DatasourceNoData`**, duración ~5 años (14/09/2026 →
  13/09/2031). Evita el aviso interno y redundante de Grafana cuando el
  datasource no devuelve datos; las reglas propias ya cubren ese caso de
  forma explícita.

> ⚠️ Este silencio, igual que las 4 reglas, **no está provisionado por
> fichero** — vive solo en la base de datos de Grafana.

---

## 7. Contact point y plantilla de email

Fichero: `grafana/provisioning/alerting/contactpoints.yaml`.

La plantilla distingue **cuatro** casos al construir el cuerpo del email:

1. **`Sensor CPD sin datos`** → mensaje dedicado, sin intentar mostrar
   ningún valor numérico.
2. **`Batería CPD baja`** → muestra el nivel en `%` (referencia `A`),
   o un texto de "no se ha podido leer" si no hay valor disponible.
3. **Temperatura/Humedad, con valor real disponible** → formato clásico
   (`Valor: X.X°C (rango normal: ...)`).
4. **Temperatura/Humedad, sin valor disponible** (`len .Values == 0`) →
   "No se han recibido lecturas de este sensor", en vez de formatear un
   `0.0` inventado.

---

## 8. Pendiente

Ver `docs/MEJORAS-FUTURAS.md`:
- Provisionar por fichero las 4 reglas y el silencio, para que sobrevivan
  a un `docker compose down -v`.
- Watchdog de systemd para el colector (sección 5.4).
- Vigilar si el "Anular tiempos" a nivel de regla (sección 2.3, 5.3) se
  mantiene estable a largo plazo, o si reaparece algún patrón de
  repetición no esperado.
