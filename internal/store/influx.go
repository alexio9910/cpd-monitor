// Paquete store: envía las lecturas de los sensores a InfluxDB.
package store

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/alexio9910/cpd-monitor/internal/config"
	"github.com/alexio9910/cpd-monitor/internal/sensor"
)

// medida es el nombre de la "measurement" de InfluxDB donde se guarda todo.
// Mantener una única measurement con tags "sensor_id"/"ubicacion" (en vez de
// una measurement por sensor) es lo que permite comparar sensores entre sí
// fácilmente en Grafana con una sola consulta.
const medida = "cpd_ambiente"

// EscritorInflux envuelve el cliente oficial de InfluxDB.
type EscritorInflux struct {
	cliente  influxdb2.Client
	writeAPI api.WriteAPIBlocking
}

// Nueva crea un escritor a partir de la configuración de conexión.
func Nueva(cfg config.InfluxDB) *EscritorInflux {
	cliente := influxdb2.NewClient(cfg.URL, cfg.Token)
	return &EscritorInflux{
		cliente:  cliente,
		writeAPI: cliente.WriteAPIBlocking(cfg.Org, cfg.Bucket),
	}
}

// Cerrar libera los recursos del cliente HTTP subyacente.
func (e *EscritorInflux) Cerrar() {
	e.cliente.Close()
}

// Comprobar valida que se puede hablar con InfluxDB (usado al arrancar,
// para fallar rápido y con un mensaje claro si la URL/token son incorrectos).
func (e *EscritorInflux) Comprobar(ctx context.Context) error {
	ok, err := e.cliente.Ping(ctx)
	if err != nil {
		return fmt.Errorf("no se pudo contactar con InfluxDB: %w", err)
	}
	if !ok {
		return fmt.Errorf("InfluxDB respondió pero no está disponible (ping negativo)")
	}
	return nil
}

// Escribir guarda una lectura de un sensor concreto.
func (e *EscritorInflux) Escribir(ctx context.Context, sensorID, ubicacion string, l sensor.Lectura) error {
	campos := map[string]interface{}{
		"temperatura_c":   float64(l.TemperaturaC),
		"humedad_rel_pct": float64(l.HumedadRelPct),
	}
	if l.BateriaPct >= 0 {
		campos["bateria_pct"] = l.BateriaPct
	}

	punto := influxdb2.NewPoint(
		medida,
		map[string]string{
			"sensor_id": sensorID,
			"ubicacion": ubicacion,
		},
		campos,
		time.Now(),
	)

	if err := e.writeAPI.WritePoint(ctx, punto); err != nil {
		return fmt.Errorf("no se pudo escribir en InfluxDB la lectura del sensor %s: %w", sensorID, err)
	}
	return nil
}
