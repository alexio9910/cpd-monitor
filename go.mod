module github.com/alexio9910/cpd-monitor

go 1.22

// Las dependencias exactas (con sus versiones y sumas de verificación) se
// generan la primera vez que compiles el proyecto en tu propia máquina,
// ejecutando:
//
//   go mod tidy
//
// Esto creará el fichero go.sum y rellenará el bloque "require" de este
// fichero automáticamente a partir de los imports usados en el código:
//
//   - tinygo.org/x/bluetooth              (lectura BLE)
//   - github.com/influxdata/influxdb-client-go/v2  (escritura en InfluxDB)
//   - gopkg.in/yaml.v3                    (lectura de config.yaml)
//
// No se fijan versiones a mano aquí a propósito: así siempre instalas la
// última versión estable disponible en el momento en que montes el
// proyecto, en lugar de una versión que yo hubiera "inventado" hoy.
