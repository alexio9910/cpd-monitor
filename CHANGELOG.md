# Changelog

Registro de cambios significativos del proyecto. Formato libre, orientado a
que cualquiera pueda entender qué cambió, por qué, y dónde mirar el detalle
técnico si hace falta.

---

## 2026-09-23

### Añadido

- **Log de eventos del colector** (`cpd_eventos` en InfluxDB): el colector
  ahora registra cada transición de conexión/desconexión BLE (no solo las
  lecturas), con mensaje descriptivo y duración de la incidencia. Visible
  en el dashboard como tabla, sin entrar por SSH. Cambios en
  `internal/store/influx.go` (nuevo método `EscribirEvento`) y
  `cmd/collector/main.go` (detección de transiciones de estado,
  `registrarTransicion`).
- **Dashboard reorganizado en secciones plegables**: "Gráficas generales",
  "Alertas" (semáforo + enlace al historial nativo de Grafana), un bloque
  por sensor, y "Log del colector".
- **Batería como panel `gauge`** (antes número plano) en los dos sensores.
- **Interruptor de anotaciones de alerta** visible en el propio dashboard
  (rayas verticales en las gráficas, activables/desactivables).
- `GF_SERVER_ROOT_URL` en `docker-compose.yml` (reutiliza
  `GRAFANA_PUBLIC_URL`): sin esto, los botones propios de Grafana
  (Silence/View dashboard/View panel) en los emails apuntaban a
  `localhost` en vez de a la IP real.
- `docs/ALERTAS.md` actualizado con el diseño final (umbrales, velocidad
  30s, el watchdog dinámico) y el historial completo de investigación de
  hoy. `README.md` actualizado en la misma línea.

### Cambiado

- **Velocidad de todas las reglas**: intervalo y confirmación (`for`)
  bajados a 30s en las 3 reglas de valor (antes 1m/1m en temperatura-humedad,
  5m/5m en batería) — de hasta 10 minutos de espera a 30-60 segundos.
- **`scripts/gestionar_sensor.py`** confirmado como sustituto definitivo de
  `anadir_sensor_dashboard.py`: gestiona `config.yaml` y los paneles del
  dashboard en un único comando (`añadir`/`quitar`), probado en ambos
  sentidos.
- Panel `annolist` ("Historial de alertas") **retirado del dashboard**:
  pese a una configuración correcta según la documentación oficial de
  Grafana y datos reales confirmados por API, el panel no los mostraba —
  fallo específico de ese tipo de panel en Grafana 13.0.2. Sustituido por
  el panel `alertlist` (que sí funciona) a ancho completo, más un enlace
  directo al historial nativo de Grafana.

### Investigado — limitación conocida, sin solución disponible

- **Agrupación de notificaciones por `sensor_id` no siempre separa el
  email.** Cuando dos sensores disparan la misma alerta en la misma
  ventana de tiempo, llegan juntos en un solo email en vez de dos
  separados. Probadas 5 vías distintas (agrupar por `sensor_id` en la
  política, combinado con `alertname`, rutas separadas por sensor, una
  alerta con huella 100% nueva para descartar estado residual, y
  `group_by` a nivel de regla individual) — todas con el mismo resultado.
  **Causa raíz encontrada** consultando la configuración viva de
  Alertmanager (no la de provisioning): Grafana inyecta una ruta interna
  automática que intercepta toda alerta y fuerza
  `group_by: ["grafana_folder", "alertname"]` antes de que la
  configuración propia llegue a aplicarse. Sin impacto funcional real —
  cuando solo un sensor tiene el problema (el caso habitual), su email
  llega individual. Detalle completo en `docs/ALERTAS.md` sección 5.7.

### Seguridad

- Contraseña de `admin` de Grafana expuesta accidentalmente en una
  conversación de soporte (además del token de InfluxDB del día
  anterior) — pendiente de rotar igual que aquel.

---

## 2026-09-22

### Corregido

- **Alertas fantasma de `-1.0°C` / `-1.0%HR`.** Las reglas de temperatura y
  humedad tenían `execErrState: Alerting`: cualquier error puntual de
  consulta a InfluxDB (por ejemplo, un `docker compose up -d` de despliegue)
  disparaba una alerta con un valor sin inicializar. Cambiado a
  `execErrState: Error` — la detección real de "sin datos" la cubre en
  exclusiva la regla dedicada, sin este efecto secundario.
- **`GRAFANA_PUBLIC_URL` desactualizada en los emails.** El enlace al
  dashboard en los correos de alerta seguía apuntando a una IP antigua
  porque `docker compose restart` no relee el `.env` — solo
  `docker compose up -d --force-recreate` lo hace.
- **La alerta de "sin datos" no cubría un sensor caído si el otro seguía
  funcionando.** La consulta original (`count()` sin filtrar por sensor)
  perdía por completo la serie de un sensor sin lecturas. Se detectó en
  caliente: el sensor 1 llevaba más de 2 horas sin emitir sin ninguna
  alerta.

### Cambiado

- **Umbrales de temperatura/humedad ampliados**: 18-33°C (antes 18-27°C) y
  25-60%HR (antes 40-60%HR).
- **Reglas de alerta provisionadas por fichero**
  (`grafana/provisioning/alerting/rules.yaml`), versionadas en Git en vez
  de vivir solo en la base de datos interna de Grafana.
- **Alerta de "sin datos" rediseñada como regla única, dinámica por
  sensor.** Calcula los minutos transcurridos desde la última lectura de
  cada sensor sobre una ventana de 30 días — cubre automáticamente
  cualquier sensor presente o futuro, sin mantenimiento manual. (Se
  evaluó primero una regla separada por sensor con un script de
  mantenimiento — descartada por innecesariamente compleja frente a esta
  solución.)
- **`scripts/anadir_sensor_dashboard.py` sustituido por
  `scripts/gestionar_sensor.py`** (`añadir`/`quitar`).

### Añadido

- **Sensor 2** añadido al sistema (`config.yaml`, panel del dashboard).
- **Paneles nuevos en el dashboard**: "Estado de alertas" (`alertlist`) e
  "Historial de alertas" (`annolist` — retirado al día siguiente, ver
  22/09).
- **Interruptor de anotaciones de alerta** en el propio dashboard.

### Seguridad

- Token de InfluxDB rotado tras haber quedado expuesto accidentalmente en
  una conversación de soporte.

### Pendiente / conocido

- El sensor 1 lleva desde el 22/09/2026 sin emitir por Bluetooth
  (0 lecturas en 2h+, repetible cada minuto) — requiere revisión física
  (pila CR2032, botón, o que esté fuera de alcance). No es un problema de
  software: el colector detecta y registra el fallo correctamente.
- Ver `docs/MEJORAS-FUTURAS.md` para el resto de mejoras evaluadas y
  pendientes (watchdog de systemd, endpoint de salud del colector, etc).
