# Mejoras futuras

Lista de mejoras razonables, ordenadas por impacto/esfuerzo, para cuando la
versión 1 esté funcionando de forma estable.

## Alta prioridad, poco esfuerzo

- **Alertas en Grafana.** Crear reglas de alerta (Alerting > Alert rules)
  sobre las mismas consultas Flux del dashboard, con umbrales basados en el
  rango recomendado por ASHRAE TC9.9 para CPD (aprox. 18-27 °C y 40-60 % HR;
  conviene revisar el rango exacto que aplique a tu instalación) y
  notificación por email, Telegram o Slack cuando se salga de rango.
- **Token de Grafana con permisos mínimos.** Ahora mismo el datasource usa el
  token de administrador de InfluxDB (`INFLUXDB_ADMIN_TOKEN`) para simplificar
  el primer arranque. En cuanto el stack esté validado, crea en InfluxDB un
  token de **solo lectura** limitado al bucket `cpd_monitorizacion` y úsalo en
  su lugar.
- **Política de retención / downsampling.** Con dos sensores leyendo cada
  60 s, el volumen de datos es pequeño, pero si en el futuro añades más
  sensores o bajas el intervalo, conviene definir una retención en el bucket
  y una tarea (`Task`) de InfluxDB que agregue a intervalos de 5-15 min los
  datos de más de unas semanas de antigüedad.
- **Exponer el estado del propio colector.** Un pequeño endpoint HTTP interno
  (`/salud`) en el colector Go que devuelva la última lectura correcta de
  cada sensor y cuándo ocurrió, para poder monitorizar "quién vigila al
  vigilante" (por ejemplo con un chequeo simple de Uptime Kuma o similar).

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
- **Redundancia de cobertura.** Dos sensores dan una foto parcial de un CPD.
  El patrón habitual en la industria es sensorizar por pasillo frío/caliente
  y por altura de rack (arriba/abajo), para detectar puntos calientes reales.
