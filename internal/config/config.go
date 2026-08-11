// Paquete config: carga la configuración del colector desde un fichero YAML.
//
// Se ha elegido YAML (en vez de variables de entorno sueltas) porque la
// configuración incluye una LISTA de sensores (cada uno con su MAC y su
// ubicación física en el CPD), y una lista se expresa de forma mucho más
// legible en YAML que en variables de entorno.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Sensor representa un sensor Sensirion SHT4x Smart Gadget físico.
type Sensor struct {
	// ID es un identificador corto y estable (ej: "pasillo_frio_a").
	// Se usa como valor del tag "sensor_id" en InfluxDB, así que conviene
	// que NUNCA cambie una vez que empieces a guardar histórico con él.
	ID string `yaml:"id"`

	// MAC es la dirección Bluetooth del gadget, formato "AA:BB:CC:DD:EE:FF".
	MAC string `yaml:"mac"`

	// Ubicacion es una descripción humana (para los paneles de Grafana).
	Ubicacion string `yaml:"ubicacion"`
}

// InfluxDB agrupa los parámetros de conexión a la base de datos.
type InfluxDB struct {
	URL    string `yaml:"url"`
	Token  string `yaml:"token"`
	Org    string `yaml:"org"`
	Bucket string `yaml:"bucket"`
}

// Config es la configuración completa del colector.
type Config struct {
	InfluxDB InfluxDB `yaml:"influxdb"`

	// IntervaloLecturaSegundos: cada cuánto se lee cada sensor.
	// 60 segundos es un buen valor por defecto para temperatura/humedad
	// ambiente en un CPD: no varían de forma brusca segundo a segundo.
	IntervaloLecturaSegundos int `yaml:"intervalo_lectura_segundos"`

	// TimeoutConexionSegundos: cuánto se espera como máximo a que el
	// gadget responda antes de darlo por "no disponible" en ese ciclo.
	TimeoutConexionSegundos int `yaml:"timeout_conexion_segundos"`

	Sensores []Sensor `yaml:"sensores"`
}

// Intervalo devuelve el intervalo de lectura como time.Duration,
// aplicando un valor por defecto sensato si no se ha configurado.
func (c Config) Intervalo() time.Duration {
	if c.IntervaloLecturaSegundos <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.IntervaloLecturaSegundos) * time.Second
}

// Timeout devuelve el timeout de conexión BLE como time.Duration.
func (c Config) Timeout() time.Duration {
	if c.TimeoutConexionSegundos <= 0 {
		return 15 * time.Second
	}
	return time.Duration(c.TimeoutConexionSegundos) * time.Second
}

// Load lee y valida el fichero YAML indicado.
func Load(path string) (Config, error) {
	var cfg Config

	datos, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("no se pudo leer el fichero de configuración %q: %w", path, err)
	}

	if err := yaml.Unmarshal(datos, &cfg); err != nil {
		return cfg, fmt.Errorf("el fichero de configuración %q no es YAML válido: %w", path, err)
	}

	if err := cfg.validar(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c Config) validar() error {
	if c.InfluxDB.URL == "" {
		return fmt.Errorf("falta influxdb.url en la configuración")
	}
	if c.InfluxDB.Token == "" {
		return fmt.Errorf("falta influxdb.token en la configuración")
	}
	if c.InfluxDB.Org == "" {
		return fmt.Errorf("falta influxdb.org en la configuración")
	}
	if c.InfluxDB.Bucket == "" {
		return fmt.Errorf("falta influxdb.bucket en la configuración")
	}
	if len(c.Sensores) == 0 {
		return fmt.Errorf("no hay ningún sensor definido bajo 'sensores' en la configuración")
	}

	vistos := map[string]bool{}
	for _, s := range c.Sensores {
		if s.ID == "" {
			return fmt.Errorf("hay un sensor sin 'id' en la configuración")
		}
		if s.MAC == "" {
			return fmt.Errorf("el sensor %q no tiene 'mac' configurada", s.ID)
		}
		if vistos[s.ID] {
			return fmt.Errorf("el id de sensor %q está repetido, debe ser único", s.ID)
		}
		vistos[s.ID] = true
	}

	return nil
}
