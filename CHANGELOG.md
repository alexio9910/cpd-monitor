# Changelog

Registro de cambios significativos del proyecto. Formato libre, orientado a
que cualquiera pueda entender qué cambió, por qué, y dónde mirar el detalle
técnico si hace falta.

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
  `docker compose up -d --force-recreate` lo hace. Documentado en
  `docs/MANUAL-EXPERTO.md`.
- **La alerta de "sin datos" no cubría un sensor caído si el otro seguía
  funcionando.** La consulta original (`count()` sin filtrar por sensor)
  perdía por completo la serie de un sensor sin lecturas — Flux ni
  siquiera devuelve una fila con valor 0 para una serie vacía, así que
  Grafana nunca entraba en estado "sin datos" mientras cualquier otro
  sensor siguiera reportando. Se detectó en caliente: el sensor 1 llevaba
  más de 2 horas sin emitir (`no se pudo localizar el sensor ... tras 15s
  de escaneo`, repetido cada minuto) sin ninguna alerta.

### Cambiado

- **Umbrales de temperatura/humedad ampliados**: 18-33°C (antes 18-27°C) y
  25-60%HR (antes 40-60%HR).
- **Reglas de alerta provisionadas por fichero**
  (`grafana/provisioning/alerting/rules.yaml`), versionadas en Git en vez
  de vivir solo en la base de datos interna de Grafana.
- **Alerta de "sin datos" rediseñada como regla única, dinámica por
  sensor.** En vez de contar lecturas en una ventana corta (10 min, que
  hacía desaparecer a un sensor caído), calcula los minutos transcurridos
  desde la última lectura de cada sensor sobre una ventana de 30 días —
  así cualquier sensor que alguna vez haya reportado sigue apareciendo en
  el resultado, con un contador que crece si deja de emitir. Cubre
  automáticamente cualquier sensor presente o futuro, sin mantenimiento
  manual, y se autolimpia sola pasados 30 días sin lecturas de un sensor
  retirado. (Se evaluó primero una regla separada por sensor con un
  script de mantenimiento — descartada por innecesariamente compleja
  frente a esta solución.)
- **`scripts/anadir_sensor_dashboard.py` sustituido por
  `scripts/gestionar_sensor.py`** (`añadir`/`quitar`), que además del
  dashboard gestiona el bloque correspondiente en `config.yaml`.

### Añadido

- **Sensor 2** añadido al sistema (`config.yaml`, panel del dashboard).
- **Paneles nuevos en el dashboard**: "Estado de alertas" (`alertlist`,
  las 4 reglas de un vistazo) e "Historial de alertas" (`annolist`,
  cronología de disparos/resoluciones), sin necesitar Loki ni tocar el
  colector.
- **Interruptor de anotaciones de alerta** en el propio dashboard (rayas
  verticales en las gráficas, activables/desactivables desde la interfaz).
- **Batería como panel `gauge`** (semicírculo con zonas de color) en vez
  de número plano, para los dos sensores.

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
