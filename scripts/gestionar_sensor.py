#!/usr/bin/env python3
"""
Gestiona TODO lo que implica añadir o quitar un sensor del proyecto:
- El bloque correspondiente en config.yaml
- Los 3 paneles de "valor actual" en el dashboard de Grafana

Lo que NO hace (porque no se puede automatizar sin más):
- Escanear la MAC real del sensor por Bluetooth (hazlo con bluetoothctl,
  ver docs/MANUAL-EXPERTO.md sección 7.1)
- Copiar los ficheros cambiados a la Raspberry Pi, reiniciar el colector
  y Grafana, ni el commit/push a Git — el script te imprime al final los
  comandos exactos que te faltan por ejecutar

Las alertas (temperatura, humedad, batería, sin-datos) NO requieren
ningún cambio: las 4 reglas ya cubren automáticamente cualquier sensor
que exista, sin necesidad de tocar nada por sensor. Ver docs/ALERTAS.md.

Uso:
    python3 scripts/gestionar_sensor.py añadir <id> <mac> "<ubicacion>"
    python3 scripts/gestionar_sensor.py quitar <id>
"""
import copy
import json
import re
import sys
from pathlib import Path

RUTA_CONFIG = Path("config.yaml")
RUTA_DASHBOARD = Path("grafana/dashboards/cpd-temp-humedad.json")
SENSOR_PLANTILLA = "sensor1"


def anadir_a_config(sensor_id: str, mac: str, ubicacion: str) -> bool:
    if not RUTA_CONFIG.exists():
        print(f"AVISO: no existe {RUTA_CONFIG} en esta carpeta — la tocas en la Pi, no en WSL. Se omite este paso.")
        return False

    texto = RUTA_CONFIG.read_text(encoding="utf-8")

    if re.search(rf'id:\s*"{re.escape(sensor_id)}"', texto):
        print(f"'{sensor_id}' ya existe en {RUTA_CONFIG}. No se ha tocado.")
        return False

    bloque_nuevo = f'  - id: "{sensor_id}"\n    mac: "{mac}"\n    ubicacion: "{ubicacion}"\n'
    texto = texto.rstrip("\n") + "\n" + bloque_nuevo
    RUTA_CONFIG.write_text(texto, encoding="utf-8")
    print(f"Añadido '{sensor_id}' a {RUTA_CONFIG}.")
    return True


def quitar_de_config(sensor_id: str) -> bool:
    if not RUTA_CONFIG.exists():
        print(f"AVISO: no existe {RUTA_CONFIG} en esta carpeta. Se omite este paso.")
        return False

    texto = RUTA_CONFIG.read_text(encoding="utf-8")
    patron = re.compile(
        rf'  - id: "{re.escape(sensor_id)}"\n(    .+\n)*',
    )
    nuevo_texto, n = patron.subn("", texto)
    if n == 0:
        print(f"'{sensor_id}' no se encontró en {RUTA_CONFIG}. No se ha tocado.")
        return False

    RUTA_CONFIG.write_text(nuevo_texto, encoding="utf-8")
    print(f"Eliminado '{sensor_id}' de {RUTA_CONFIG}.")
    return True


def anadir_paneles_dashboard(sensor_id: str, ubicacion: str) -> bool:
    with open(RUTA_DASHBOARD, encoding="utf-8") as f:
        dashboard = json.load(f)

    paneles = dashboard["panels"]

    for panel in paneles:
        for target in panel.get("targets", []):
            if f'r.sensor_id == "{sensor_id}"' in target.get("query", ""):
                print(f"Ya existen paneles para '{sensor_id}' en el dashboard. No se ha tocado.")
                return False

    plantillas = [
        p for p in paneles
        if p["type"] in ("stat", "gauge")
        and any(f'r.sensor_id == "{SENSOR_PLANTILLA}"' in t.get("query", "") for t in p.get("targets", []))
    ]

    if len(plantillas) != 3:
        print(f"AVISO: se esperaban 3 paneles plantilla de '{SENSOR_PLANTILLA}' (stat/gauge), "
              f"se encontraron {len(plantillas)}. Revisa el dashboard a mano.")
        return False

    filas_existentes_y = {p["gridPos"]["y"] for p in paneles if p["type"] in ("stat", "gauge")}
    siguiente_id = max(p["id"] for p in paneles) + 1
    fila_y = max(y + 6 for y in filas_existentes_y)

    nuevos_paneles = []
    for plantilla in plantillas:
        nuevo = copy.deepcopy(plantilla)
        nuevo["id"] = siguiente_id
        siguiente_id += 1
        nuevo["gridPos"]["y"] = fila_y
        nuevo["title"] = re.sub(r" — Sensor 1$", "", nuevo["title"]) + f" — {ubicacion}"
        for target in nuevo["targets"]:
            target["query"] = target["query"].replace(
                f'r.sensor_id == "{SENSOR_PLANTILLA}"',
                f'r.sensor_id == "{sensor_id}"',
            )
        nuevos_paneles.append(nuevo)

    dashboard["panels"].extend(nuevos_paneles)
    dashboard["version"] = dashboard.get("version", 1) + 1

    with open(RUTA_DASHBOARD, "w", encoding="utf-8") as f:
        json.dump(dashboard, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print(f"Añadidos {len(nuevos_paneles)} paneles para '{sensor_id}' ({ubicacion}) al dashboard.")
    return True


def quitar_paneles_dashboard(sensor_id: str) -> bool:
    with open(RUTA_DASHBOARD, encoding="utf-8") as f:
        dashboard = json.load(f)

    antes = len(dashboard["panels"])
    dashboard["panels"] = [
        p for p in dashboard["panels"]
        if not any(f'r.sensor_id == "{sensor_id}"' in t.get("query", "") for t in p.get("targets", []))
    ]
    quitados = antes - len(dashboard["panels"])

    if quitados == 0:
        print(f"No se encontraron paneles de '{sensor_id}' en el dashboard. No se ha tocado.")
        return False

    dashboard["version"] = dashboard.get("version", 1) + 1

    with open(RUTA_DASHBOARD, "w", encoding="utf-8") as f:
        json.dump(dashboard, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print(f"Quitados {quitados} paneles de '{sensor_id}' del dashboard.")
    return True


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)

    accion = sys.argv[1]

    if accion == "añadir":
        if len(sys.argv) != 5:
            print('Uso: python3 scripts/gestionar_sensor.py añadir <id> <mac> "<ubicacion>"')
            sys.exit(1)
        sensor_id, mac, ubicacion = sys.argv[2], sys.argv[3], sys.argv[4]

        anadir_a_config(sensor_id, mac, ubicacion)
        anadir_paneles_dashboard(sensor_id, ubicacion)

        print()
        print("Hecho en tu copia local. Ahora falta, en este orden:")
        print(f"  1. Si config.yaml se ha tocado aquí: scp config.yaml cpd@192.168.169.5:~/cpd-monitor/config.yaml")
        print(f"     ssh cpd@192.168.169.5 'sudo cp ~/cpd-monitor/config.yaml /opt/cpd-monitor/config.yaml && "
              f"sudo chown cpdmonitor:cpdmonitor /opt/cpd-monitor/config.yaml && "
              f"sudo systemctl restart cpd-monitor'")
        print(f"  2. git add -A && git commit -m \"Añade sensor {sensor_id} ({ubicacion})\" && git push")
        print(f"  3. ./deploy.sh   (o: en la Pi, git pull && docker compose restart grafana)")
        print("Las alertas no requieren ningún cambio: ya cubren a este sensor automáticamente.")

    elif accion == "quitar":
        if len(sys.argv) != 3:
            print("Uso: python3 scripts/gestionar_sensor.py quitar <id>")
            sys.exit(1)
        sensor_id = sys.argv[2]

        quitar_de_config(sensor_id)
        quitar_paneles_dashboard(sensor_id)

        print()
        print("Hecho en tu copia local. Ahora falta, en este orden:")
        print(f"  1. Si config.yaml se ha tocado aquí: scp config.yaml cpd@192.168.169.5:~/cpd-monitor/config.yaml")
        print(f"     ssh cpd@192.168.169.5 'sudo cp ~/cpd-monitor/config.yaml /opt/cpd-monitor/config.yaml && "
              f"sudo chown cpdmonitor:cpdmonitor /opt/cpd-monitor/config.yaml && "
              f"sudo systemctl restart cpd-monitor'")
        print(f"  2. git add -A && git commit -m \"Quita sensor {sensor_id}\" && git push")
        print(f"  3. ./deploy.sh   (o: en la Pi, git pull && docker compose restart grafana)")
        print(f"La alerta de 'sin datos' de este sensor se autolimpiará sola pasados 30 días sin lecturas suyas.")

    else:
        print(f"Acción desconocida: '{accion}'. Usa 'añadir' o 'quitar'.")
        sys.exit(1)


if __name__ == "__main__":
    main()
