#!/usr/bin/env bash
# Actualiza y reinicia el proyecto en la Raspberry Pi desde tu máquina de
# desarrollo (WSL). Automatiza: git pull + reconstruir stack + recompilar
# + reiniciar servicio, todo en un solo comando.
#
# Uso:
#   ./deploy.sh
#
# Requiere una variable PI_HOST en tu .env local (el de tu WSL, no hace
# falta que tenga el resto de variables de InfluxDB/Grafana), por ejemplo:
#   PI_HOST=cpd@192.168.99.82
# Así el host real nunca queda escrito en un fichero que se sube a GitHub.

set -e

if [ -f .env ]; then
  export "$(grep -E '^PI_HOST=' .env | xargs)"
fi

if [ -z "$PI_HOST" ]; then
  echo "ERROR: falta PI_HOST en tu .env. Añade una línea como:"
  echo "  PI_HOST=usuario@IP_DE_TU_RASPBERRY"
  exit 1
fi

echo "==> Desplegando en ${PI_HOST}..."

ssh "$PI_HOST" '
  set -e
  cd cpd-monitor &&
  echo "-- git pull --" &&
  git pull &&
  echo "-- docker compose up -d (por si cambio algo del stack) --" &&
  docker compose up -d &&
  echo "-- make build --" &&
  make build &&
  echo "-- copiando binario a /opt/cpd-monitor --" &&
  sudo cp bin/cpd-monitor /opt/cpd-monitor/ &&
  echo "-- reiniciando servicio --" &&
  sudo systemctl restart cpd-monitor &&
  sleep 3 &&
  sudo systemctl status cpd-monitor --no-pager
'

echo "==> Despliegue completado."
