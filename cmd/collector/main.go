// Comando collector: lee periódicamente los sensores Sensirion SHT4x Smart
// Gadget configurados y escribe sus valores en InfluxDB.
//
// Uso:
//
//	collector -config config.yaml
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alexio9910/cpd-monitor/internal/config"
	"github.com/alexio9910/cpd-monitor/internal/sensor"
	"github.com/alexio9910/cpd-monitor/internal/store"
)

func main() {
	rutaConfig := flag.String("config", "config.yaml", "ruta al fichero de configuración YAML")
	flag.Parse()

	cfg, err := config.Load(*rutaConfig)
	if err != nil {
		log.Fatalf("error de configuración: %v", err)
	}

	if err := sensor.Habilitar(); err != nil {
		log.Fatalf("error de Bluetooth: %v", err)
	}

	escritor := store.Nueva(cfg.InfluxDB)
	defer escritor.Cerrar()

	ctxArranque, cancelArranque := context.WithTimeout(context.Background(), 10*time.Second)
	if err := escritor.Comprobar(ctxArranque); err != nil {
		cancelArranque()
		log.Fatalf("no se pudo conectar con InfluxDB al arrancar: %v", err)
	}
	cancelArranque()

	log.Printf("colector iniciado: %d sensor(es), lectura cada %s", len(cfg.Sensores), cfg.Intervalo())

	// Contexto que se cancela al recibir SIGINT/SIGTERM (Ctrl+C, o
	// "systemctl stop cpd-monitor"), para poder parar de forma ordenada.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// mu serializa TODAS las operaciones BLE (ver comentario en
	// internal/sensor/gadget.go): aunque cada sensor tiene su propia
	// gorutina y su propio ticker, solo uno a la vez usa el adaptador.
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, s := range cfg.Sensores {
		wg.Add(1)
		go func(s config.Sensor) {
			defer wg.Done()
			cicloSensor(ctx, s, cfg, escritor, &mu)
		}(s)
	}

	wg.Wait()
	log.Println("colector detenido")
}

// cicloSensor lee un sensor de forma periódica hasta que ctx se cancele.
func cicloSensor(ctx context.Context, s config.Sensor, cfg config.Config, escritor *store.EscritorInflux, mu *sync.Mutex) {
	ticker := time.NewTicker(cfg.Intervalo())
	defer ticker.Stop()

	// Primera lectura inmediata, sin esperar al primer tick.
	leerYGuardar(ctx, s, cfg, escritor, mu)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			leerYGuardar(ctx, s, cfg, escritor, mu)
		}
	}
}

func leerYGuardar(ctx context.Context, s config.Sensor, cfg config.Config, escritor *store.EscritorInflux, mu *sync.Mutex) {
	mu.Lock()
	lectura, err := sensor.Leer(s.MAC, cfg.Timeout())
	mu.Unlock()

	if err != nil {
		// No se para el colector por un fallo puntual de un sensor: se
		// registra el error y se reintenta en el siguiente ciclo. Esto es
		// clave para un CPD, donde perder una lectura no debe tirar abajo
		// el resto de la monitorización.
		log.Printf("[%s] error leyendo el sensor: %v", s.ID, err)
		return
	}

	ctxEscritura, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := escritor.Escribir(ctxEscritura, s.ID, s.Ubicacion, lectura); err != nil {
		log.Printf("[%s] error escribiendo en InfluxDB: %v", s.ID, err)
		return
	}

	log.Printf("[%s] %.2f°C  %.1f%%HR  bateria=%d%%", s.ID, lectura.TemperaturaC, lectura.HumedadRelPct, lectura.BateriaPct)
}
