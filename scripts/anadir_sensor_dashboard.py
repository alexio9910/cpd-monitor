#!/usr/bin/env python3
"""
Añade al dashboard de Grafana los tres paneles de "valor actual"
(Temperatura, Humedad, Batería) para un sensor nuevo, duplicando los del
sensor plantilla (sensor1) y ajustando título, filtro de la consulta y
posición en la rejilla automáticamente.

Uso:
    python3 scripts/anadir_sensor_dashboard.py <sensor_id> "<Ubicación legible>"

Ejemplo:
    python3 scripts/anadir_sensor_dashboard.py sensor2 "Sensor 2"

Tras ejecutarlo, reinicia Grafana para que recoja el cambio:
    docker compose restart grafana

Es seguro ejecutarlo varias veces: si detecta que ya existen paneles para
ese sensor_id, no hace nada y avisa, en vez de duplicarlos.
"""
import copy
import json
import sys

RUTA_DASHBOARD = "grafana/dashboards/cpd-temp-humedad.json"
SENSOR_PLANTILLA = "sensor1"


def main():
    if len(sys.argv) != 3:
        print(f'Uso: python3 {sys.argv[0]} <sensor_id> "<Ubicacion legible>"')
        print(f'Ejemplo: python3 {sys.argv[0]} sensor2 "Sensor 2"')
        sys.exit(1)

    nuevo_id_sensor = sys.argv[1]
    nueva_ubicacion = sys.argv[2]

    with open(RUTA_DASHBOARD, encoding="utf-8") as f:
        dashboard = json.load(f)

    paneles = dashboard["panels"]

    # Evita duplicados si ya se ejecuto antes para este mismo sensor
    for panel in paneles:
        for target in panel.get("targets", []):
            if f'r.sensor_id == "{nuevo_id_sensor}"' in target.get("query", ""):
                print(f"Ya existen paneles para '{nuevo_id_sensor}'. No se ha hecho nada.")
                sys.exit(0)

    # Localiza los tres paneles plantilla (los del sensor1)
    plantillas = [
        p for p in paneles
        if any(f'r.sensor_id == "{SENSOR_PLANTILLA}"' in t.get("query", "") for t in p.get("targets", []))
    ]

    if len(plantillas) != 3:
        print(f"AVISO: se esperaban 3 paneles plantilla de '{SENSOR_PLANTILLA}', "
              f"se encontraron {len(plantillas)}. Revisa el dashboard a mano.")
        sys.exit(1)

    siguiente_id = max(p["id"] for p in paneles) + 1
    fila_y = max(p["gridPos"]["y"] + p["gridPos"]["h"] for p in paneles)

    nuevos_paneles = []
    for plantilla in plantillas:
        nuevo = copy.deepcopy(plantilla)
        nuevo["id"] = siguiente_id
        siguiente_id += 1
        nuevo["gridPos"]["y"] = fila_y
        nuevo["title"] = nuevo["title"].split(" — Sensor 1")[0] + f" — {nueva_ubicacion}"
        for target in nuevo["targets"]:
            target["query"] = target["query"].replace(
                f'r.sensor_id == "{SENSOR_PLANTILLA}"',
                f'r.sensor_id == "{nuevo_id_sensor}"',
            )
        nuevos_paneles.append(nuevo)

    dashboard["panels"].extend(nuevos_paneles)

    with open(RUTA_DASHBOARD, "w", encoding="utf-8") as f:
        json.dump(dashboard, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print(f"Añadidos {len(nuevos_paneles)} paneles para '{nuevo_id_sensor}' ({nueva_ubicacion}).")
    print("Ahora ejecuta: docker compose restart grafana")


if __name__ == "__main__":
    main()
