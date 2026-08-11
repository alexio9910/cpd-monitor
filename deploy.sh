#!/usr/bin/env bash
# Actualiza y reinicia el proyecto en la Raspberry Pi desde tu WSL.
#
# Uso:
#   ./deploy.sh
#
# La primera vez, edita la línea PI_HOST de más abajo con el usuario e IP
# reales de tu Raspberry Pi (ej: "pi@192.168.1.50").

set -e

PI_HOST="tuusuario@IP_DE_LA_PI"

echo "==> Desplegando en ${PI_HOST}..."

ssh "$PI_HOST" '
  set -e
  cd cpd-monitor &&
  echo "-- git pull --" &&
  git pull &&
  echo "-- docker compose up -d --" &&
  docker compose up -d &&
  echo "-- make build --" &&
  make build &&
  echo "-- copiando binario a /opt/cpd-monitor --" &&
  sudo cp bin/cpd-monitor /opt/cpd-monitor/ &&
  echo "-- reiniciando servicio --" &&
  sudo systemctl restart cpd-monitor &&
  sudo systemctl status cpd-monitor --no-pager
'

echo "==> Despliegue completado."
