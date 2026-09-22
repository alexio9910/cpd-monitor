# Mejoras futuras

Lista de mejoras razonables, ordenadas por impacto/esfuerzo, para cuando la
versión 1 esté funcionando de forma estable.

## Alta prioridad, poco esfuerzo

- ~~**Corregir el modo de reducción en las reglas de alerta (evita un falso
  `-1.0`).**~~ ✅ Resuelto el 14/09/2026 (modo `dropNN`). Ver
  `docs/ALERTAS.md` sección 5.1.
- ~~**Añadir margen de histéresis a las reglas de alerta (evita el
  parpadeo).**~~ ✅ Resuelto el 16/09/2026 — periodo pendiente de 1m en
  temperatura/humedad, grupos de evaluación propios (`CPD-Ambiente`, 1m en
  vez de los 10s genéricos originales), y corregido además un segundo bug
  relacionado ("Faltan evaluaciones de series para resolver" en su valor
  por defecto de 2, causaba resoluciones falsas con huecos BLE puntuales
  del colector). Ver `docs/ALERTAS.md` sección 5.2.
- ~~**Alertas en Grafana.**~~ ✅ Implementado — cuatro reglas: temperatura
  (18-33°C) y humedad (25-60%HR) con aviso inmediato y resolución
  automática; `Sensor CPD sin datos` con un único aviso por incidencia
  (24h, ajustado a nivel de regla) y recuperación inmediata; `Batería CPD
  baja` (<10%). Mensajes diferenciados por tipo de alerta. Detalle
  completo en `docs/ALERTAS.md` y `docs/MANUAL-EXPERTO.md` sección 3.
- ~~**Token de Grafana con permisos mínimos.**~~ ✅ Implementado — el
  datasource usa un token de InfluxDB de solo lectura
  (`GRAFANA_INFLUXDB_TOKEN`), no el de administrador. Ver
  `docs/MANUAL-EXPERTO.md` sección 6.
- ~~**Provisionar por fichero las reglas de alerta.**~~ ✅ Implementado
  (24/09/2026) — las 4 reglas viven ahora en
  `grafana/provisioning/alerting/rules.yaml`, versionadas en Git, y
  sobreviven a un `docker compose down -v` en vez de tener que
  recrearlas a mano en la interfaz. De paso se corrigió un segundo fallo
  distinto al de la sección 5.1 de `docs/ALERTAS.md`: las reglas de
  temperatura/humedad tenían `execErrState: Alerting`, lo que provocaba
  alertas fantasma con valores `-1.0` cuando InfluxDB tenía un hipo
  momentáneo (por ejemplo durante un `docker compose up -d` de
  despliegue), incluso con el `dropNN` ya corregido. Ahora usan
  `execErrState: Error` (no alerta por un fallo puntual de consulta) y
  la detección de "colector caído / sin datos" la cubre en exclusiva la
  regla dedicada basada en `count()`, sin redundancia ni falsos
  positivos. El silencio (`mute timing`) de `DatasourceNoData`, si
  sigue en uso, queda pendiente de provisionar igual (`mute-timings.yaml`).
- ~~**Panel de estado de alertas en el dashboard.**~~ ✅ Implementado
  (24/09/2026) — dos paneles nativos de Grafana (sin instalar nada
  extra): "Estado de alertas" (`alertlist`, semáforo de las 4 reglas de
  un vistazo) e "Historial de alertas" (`annolist`, tabla cronológica de
  disparos y resoluciones). Así se puede ver qué ha pasado sin entrar a
  la Raspberry Pi.
- ~~**La alerta de "sin datos" no detectaba un sensor caído si otro
  seguía funcionando.**~~ ✅ Corregido (22/09/2026) — descubierto en
  caliente: el sensor 1 llevó más de 2 horas sin emitir sin generar
  ninguna alerta, porque la consulta original (`count()` sobre todos los
  sensores juntos) hacía desaparecer por completo la serie de un sensor
  sin lecturas en vez de mostrar un recuento de 0. Rediseñada como una
  única regla que calcula los minutos transcurridos desde la última
  lectura de cada sensor (ventana de 30 días en vez de 10 minutos): así
  cualquier sensor que alguna vez haya reportado sigue devolviendo una
  fila, cubriendo automáticamente cualquier sensor presente o futuro sin
  mantenimiento manual. Detalle en `CHANGELOG.md` (22/09/2026).
- ~~**Gestión de sensores en un solo paso.**~~ ✅ Implementado
  (22/09/2026) — `scripts/anadir_sensor_dashboard.py` sustituido por
  `scripts/gestionar_sensor.py añadir|quitar`, que además del dashboard
  gestiona el bloque de `config.yaml` en un único comando, e imprime al
  terminar los pasos que faltan (copiar a la Pi, commit/push).
- **Alertas también por Telegram/Slack.** El contact point de email ya está
  provisionado (`grafana/provisioning/alerting/`); añadir un segundo
  "receiver" del mismo contact point (o uno nuevo) para otro canal es
  sencillo y no requiere tocar las reglas de alerta.
- **Watchdog de systemd para el colector.** El 14-16/09/2026 el proceso
  `cpd-monitor` quedó bloqueado casi 2 días (`active (running)` en
  `systemctl status`, pero sin escribir ningún log nuevo) sin que systemd
  lo detectara ni reiniciara — se descubrió por casualidad comparando el
  dashboard con un email que no cuadraba. Un watchdog
  (`WatchdogSec=` en el `.service` + que el colector llame a
  `sd_notify(WATCHDOG=1)` en cada ciclo exitoso, vía
  `github.com/coreos/go-systemd/daemon`) haría que systemd lo reiniciara
  solo en minutos si deja de responder, sin depender de que alguien lo
  note manualmente ni de esperar los 10 min de la regla "sin datos". Ver
  `docs/ALERTAS.md` sección 5.4.
- **Política de retención / downsampling.** Con dos sensores leyendo cada
  60 s, el volumen de datos es pequeño, pero si en el futuro añades más
  sensores o bajas el intervalo, conviene definir una retención en el bucket
  y una tarea (`Task`) de InfluxDB que agregue a intervalos de 5-15 min los
  datos de más de unas semanas de antigüedad.
- **Exponer el estado del propio colector.** Un pequeño endpoint HTTP interno
  (`/salud`) en el colector Go que devuelva la última lectura correcta de
  cada sensor y cuándo ocurrió — complementaría bien al watchdog de
  systemd de arriba, permitiendo monitorizar el colector también desde
  fuera (por ejemplo Uptime Kuma).
- **Log de eventos propio en InfluxDB (`cpd_eventos`).** El colector podría
  escribir un evento cada vez que un sensor BLE se conecta/desconecta
  (no solo cuando salta una alerta de Grafana), con mensajes en texto
  claro. Da un histórico técnico más rico y permanente que las anotaciones
  de Grafana, pero requiere tocar `internal/sensor/gadget.go` y
  `cmd/collector/main.go`, recompilar y redesplegar el binario. Evaluado
  y descartado por ahora a favor de los paneles nativos de Grafana
  (`alertlist`/`annolist`), que cubren la misma necesidad sin tocar el
  colector.

## Media prioridad

- **CI más completo.** El workflow incluido (`.github/workflows/build.yml`)
  compila y valida el código en cada push. Se le puede añadir `golangci-lint`
  y, si se quiere, un job que valide con `docker compose config` que el
  `docker-compose.yml` es válido.
- **Alcance/fiabilidad del BLE dentro del CPD.** Los racks metálicos y las
  puertas de armario atenúan mucho la señal BLE. Si algún sensor queda
  "fuera de rango" de forma habitual, dos opciones: (a) acercar el host que
  ejecuta el colector, o usar un adaptador USB Bluetooth con antena externa
  en vez del Bluetooth integrado; (b) sustituir la lectura BLE directa por un
  pequeño gateway (p. ej. un ESP32 barato) colocado junto a cada sensor, que
  lea el gadget por BLE y publique el valor por Wi-Fi/Ethernet (MQTT) hacia
  el colector — a costa de más piezas que mantener.
- **Contenerizar el colector.** Es factible metiéndolo en Docker con
  `network_mode: host` y montando `/var/run/dbus` y el socket de BlueZ, pero
  se ha dejado fuera de la v1 a propósito (ver
  [docs/ARQUITECTURA.md](ARQUITECTURA.md)) por la complejidad de permisos que
  añade sin necesidad real todavía.

## Baja prioridad / a vigilar

- **InfluxDB 3 Core.** Es el sucesor de InfluxDB 2.x y motor por defecto en
  las imágenes Docker "latest" desde mediados de septiembre de 2026. Hoy por
  hoy limita las consultas a un rango de 72 horas en su edición gratuita, lo
  que lo hace poco práctico para el histórico de meses que interesa en un
  CPD. Si esa limitación desaparece o deja de importarte (por ejemplo, si
  solo necesitas los últimos días y archivas el resto aparte), migrar es
  razonable: el formato de escritura (line protocol) es compatible.
- **Telegraf.** Es la herramienta "oficial" de InfluxData para recolectar
  métricas de mil fuentes distintas sin programar nada, pero no tiene (a
  fecha de hoy) un plugin de entrada para el perfil BLE propietario de
  Sensirion, así que no sustituye al colector en Go. Puede ser útil el día
  de mañana si añades otras fuentes de datos al mismo InfluxDB (por ejemplo,
  métricas de los propios racks/PDUs).
- **Volver a `tinygo.org/x/bluetooth` si arreglan su compatibilidad con
  BlueZ reciente.** Se descartó esa librería de abstracción por un bug sin
  resolver a día de hoy con BlueZ ≥5.55 (ver
  [issue #118](https://github.com/tinygo-org/bluetooth/issues/118)); el
  colector habla con BlueZ directamente por D-Bus en su lugar (ver
  `docs/ARQUITECTURA.md`). Si en el futuro se publica una versión estable
  con el arreglo, valdría la pena reevaluar: simplificaría bastante
  `internal/sensor/gadget.go`.
- **Redundancia de cobertura.** Dos sensores dan una foto parcial de un CPD.
  El patrón habitual en la industria es sensorizar por pasillo frío/caliente
  y por altura de rack (arriba/abajo), para detectar puntos calientes reales.
