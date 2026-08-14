#!/bin/sh
# Espera hasta que InfluxDB responda a su endpoint de salud antes de
# arrancar el colector. Evita que systemd cuente un arranque en frío
# (justo tras reiniciar el host) como un fallo real del servicio.
#
# Instalación: copiar a /opt/cpd-monitor/wait-for-influxdb.sh, dar
# permisos de ejecución, y referenciarlo con ExecStartPre en
# cpd-monitor.service (ya viene configurado así de fábrica).

INTENTOS=30
ESPERA_SEGUNDOS=2

for i in $(seq 1 "$INTENTOS"); do
  if curl -sf http://localhost:8086/health >/dev/null 2>&1; then
    exit 0
  fi
  sleep "$ESPERA_SEGUNDOS"
done

echo "InfluxDB no respondio tras $((INTENTOS * ESPERA_SEGUNDOS))s" >&2
exit 1
