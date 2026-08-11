module github.com/alexio9910/cpd-monitor

go 1.23.8

toolchain go1.24.4

// Las dependencias exactas (con sus versiones y sumas de verificación) se
// generan la primera vez que compiles el proyecto en tu propia máquina,
// ejecutando:
//
//   go mod tidy
//
// Esto creará el fichero go.sum y rellenará el bloque "require" de este
// fichero automáticamente a partir de los imports usados en el código:
//
//   - github.com/godbus/dbus/v5           (comunicación BLE con BlueZ,
//                                           hablando directamente por D-Bus)
//   - github.com/influxdata/influxdb-client-go/v2  (escritura en InfluxDB)
//   - gopkg.in/yaml.v3                    (lectura de config.yaml)
//
// No se fijan versiones a mano aquí a propósito: así siempre instalas la
// última versión estable disponible en el momento en que montes el
// proyecto, en lugar de una versión que yo hubiera "inventado" hoy.

require (
	github.com/godbus/dbus/v5 v5.1.0
	github.com/influxdata/influxdb-client-go/v2 v2.14.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/oapi-codegen/runtime v1.0.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
)
