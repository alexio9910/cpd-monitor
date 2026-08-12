# Guía desde cero — Monitorización del CPD

Esta guía asume que no has hecho nunca nada parecido. Vas a montar, paso a
paso, un sistema que lee dos sensores de temperatura/humedad por Bluetooth y
te los enseña en gráficas bonitas — todo alojado en una Raspberry Pi dentro
de tu CPD.

Si en algún punto un comando te da un error, **para ahí, no sigas** a la
siguiente instrucción: cópiame el error exacto y lo resolvemos antes de
continuar. Casi todos los errores en este tipo de proyectos se resuelven en
un minuto si se atajan en el momento en que aparecen.

> 💡 **La forma más cómoda de seguir esta guía es dentro de Claude Desktop,
> usando Claude Code**, apuntando a tu carpeta de WSL. Así, en vez de copiar
> y pegar cada comando tú mismo, puedes pedirle a Claude que los ejecute por
> ti, uno a uno, y que te explique qué hace cada uno antes de lanzarlo. Al
> final de esta guía (Fase 10) te explico cómo arrancar esa sesión.

---

## Fase 0 — Glosario de 2 minutos

No hace falta memorizar esto, solo tenerlo a mano para cuando aparezca un
término y no sepas qué es.

| Término | Qué es, en corto |
|---|---|
| **Terminal / consola** | La ventana negra donde escribes comandos de texto en vez de hacer clic. En Windows, tú usas **WSL** para tener una terminal de Linux. |
| **WSL** | "Windows Subsystem for Linux": un Linux completo funcionando dentro de tu Windows. Ya lo tienes instalado. |
| **Git** | Un programa que guarda el *historial de cambios* de tu proyecto (quién cambió qué y cuándo). Cada "guardado" se llama **commit**. |
| **GitHub** | Una web donde alojas tu proyecto de Git, para tener copia de seguridad y poder descargarlo desde cualquier sitio (como la Raspberry Pi). |
| **Repositorio (repo)** | La carpeta de tu proyecto, tal y como la ve Git/GitHub. |
| **Push / Pull** | *Push* = subir tus commits de tu ordenador a GitHub. *Pull* = bajar a tu ordenador (o a la Pi) los commits que hay en GitHub. |
| **SSH** | Una forma segura de conectarte a otro ordenador por terminal (lo usarás para entrar en la Raspberry Pi) o de identificarte ante GitHub sin escribir tu contraseña cada vez. |
| **Docker** | Un programa que ejecuta aplicaciones "empaquetadas" (contenedores) sin tener que instalarlas una a una a mano. Lo usarás para InfluxDB y Grafana. |
| **Go** | El lenguaje de programación en el que está escrito el programa que lee los sensores. "Compilar" = convertir ese código en un programa ejecutable. |
| **BLE (Bluetooth Low Energy)** | El tipo de Bluetooth de bajo consumo que usan tus sensores Sensirion. |
| **InfluxDB** | La base de datos donde se guardan las lecturas de temperatura/humedad, una detrás de otra en el tiempo. |
| **Grafana** | La aplicación web que dibuja esos datos como gráficas. |
| **systemd / servicio** | El sistema de Linux que hace que un programa arranque solo al encender el dispositivo y se reinicie solo si falla. |

---

## Fase 1 — Comprobar/instalar las herramientas en tu WSL

Abre tu terminal de WSL y ejecuta, una a una, estas comprobaciones:

```bash
git --version
go version
docker --version
docker compose version
```

Si **todas** te devuelven un número de versión, salta a la Fase 2. Si alguna
dice "command not found", instálala:

```bash
sudo apt update

# Si falta git:
sudo apt install -y git

# Si falta Go:
sudo apt install -y golang

# Si falta Docker (incluye el plugin "compose"):
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# cierra la terminal de WSL y ábrela de nuevo para que el cambio de grupo aplique
```

---

## Fase 2 — Crear la carpeta del proyecto

```bash
mkdir -p ~/proyectos/cpd-monitor
cd ~/proyectos/cpd-monitor
```

Ahora copia dentro de esa carpeta **todos** los ficheros del `.zip` que te
he generado (descomprímelo directamente ahí, o pídele a Claude Code que lo
haga por ti si estás trabajando desde Claude Desktop).

Comprueba que está todo:

```bash
ls -la
```

Deberías ver, entre otras cosas: `README.md`, `go.mod`, `docker-compose.yml`,
`cmd/`, `internal/`, `grafana/`, `docs/`.

---

## Fase 3 — Qué es cada fichero (para que no copies y pegues a ciegas)

| Fichero / carpeta | Para qué sirve |
|---|---|
| `cmd/collector/main.go` | El "arranque" del programa que lee los sensores. |
| `internal/sensor/` | El código que sabe hablar por Bluetooth con tus sensores Sensirion. |
| `internal/store/` | El código que guarda las lecturas en InfluxDB. |
| `internal/config/` | El código que lee tu fichero `config.yaml`. |
| `config.example.yaml` | Plantilla de configuración. La copiarás a `config.yaml` y pondrás tus datos reales. |
| `docker-compose.yml` | Le dice a Docker qué aplicaciones levantar (InfluxDB y Grafana) y cómo. |
| `.env.example` | Plantilla de contraseñas/tokens para `docker-compose.yml`. La copiarás a `.env`. |
| `grafana/` | El "panel de control" ya diseñado, para que Grafana lo cargue solo. |
| `deploy/systemd/cpd-monitor.service` | La configuración para que el colector arranque solo con la Raspberry Pi. |
| `deploy.sh` | Un atajo: un solo comando que actualiza todo en la Raspberry Pi. |
| `docs/ARQUITECTURA.md` | Por qué se ha construido así el proyecto (para cuando quieras entender el "por qué"). |
| `docs/MEJORAS-FUTURAS.md` | Ideas para seguir mejorando el proyecto más adelante. |

**Nunca edites a mano** los ficheros `config.yaml` o `.env` en el propio
repositorio de GitHub — esos dos contienen contraseñas y **no se suben nunca**
(ya está configurado así en `.gitignore`, no tienes que hacer nada para eso).

---

## Fase 4 — Subir el proyecto a GitHub

### 4.1 — Convertir la carpeta en un repositorio Git

```bash
cd ~/proyectos/cpd-monitor
git init
git branch -M main
```

### 4.2 — Decirle a Git quién eres (solo la primera vez que usas Git en esta máquina)

```bash
git config --global user.name "Tu Nombre"
git config --global user.email "tu-email@ejemplo.com"
```

### 4.3 — Crear tu clave SSH para GitHub (si no la tienes ya)

```bash
ls ~/.ssh/id_ed25519.pub 2>/dev/null && echo "Ya tienes una clave" || ssh-keygen -t ed25519 -C "tu-email@ejemplo.com"
```

Si el comando anterior generó una clave nueva, muéstrala y cópiala:

```bash
cat ~/.ssh/id_ed25519.pub
```

Ve a GitHub → tu foto de perfil → **Settings** → **SSH and GPG keys** →
**New SSH key**, y pega ahí el contenido que has copiado.

### 4.4 — Crear el repositorio vacío en GitHub

Desde la web de GitHub: entra en tu cuenta (o en tu organización, si
usas una) → **New repository** → nómbralo `cpd-monitor` → **no** marques
"Add a README" (ya tienes uno) → crear.

### 4.5 — El primer commit y el primer push

```bash
git add .
git commit -m "Estructura inicial del proyecto: colector Go, docker-compose y dashboard de Grafana"

git remote add origin git@github.com:alexio9910/cpd-monitor.git
git push -u origin main
```

> 💡 Si vas a reutilizar este proyecto como base para otro repositorio
> distinto, cambia `alexio9910` por tu propio usuario de GitHub (o el
> nombre de tu organización, si usas una) en estos mismos sitios: aquí,
> en `go.mod` (primera línea), y en `internal/store/influx.go` /
> `cmd/collector/main.go` (los imports que empiezan por
> `github.com/alexio9910/...`).

Refresca la página del repositorio en GitHub: deberías ver todos tus
ficheros ahí. 🎉

---

## Fase 5 — Identificar tus sensores Bluetooth

Hay dos formas de conseguir la dirección MAC de cada sensor. Elige la que
te resulte más cómoda.

### Opción A — con el móvil, usando nRF Connect (solo Android)

1. Enciende el sensor Sensirion SHT4x Smart Gadget.
2. Instala la app **nRF Connect** en tu móvil (gratuita).
3. Pulsa **Scan**, busca el dispositivo (algo como "SHT4x Smart Gadget" o
   "SHT40 Gadget") y anota su **dirección MAC** (formato
   `AA:BB:CC:DD:EE:FF`).

> ⚠️ **Si usas iPhone, esta opción no funciona**: iOS oculta la dirección
> MAC real a todas las apps (restricción de privacidad de Apple), ni
> siquiera nRF Connect la puede leer ahí. Usa la Opción B.

### Opción B — escaneando desde la propia Raspberry Pi (recomendada)

Esta opción necesita que ya hayas instalado `bluez` en la Pi (Fase 6.3, un
poco más abajo) — si todavía no has llegado ahí, completa primero la Fase
6 hasta el punto 6.3 y vuelve aquí.

Con el sensor encendido y cerca de la Pi:

```bash
ssh pi@IP_DE_LA_RASPBERRY   # o el usuario que hayas configurado
bluetoothctl
scan on
```

Espera 15-20 segundos y busca una línea con un nombre parecido a "SHT4x
Smart Gadget" o "SHT40 Gadget" — la MAC aparece justo al lado. Luego:

```
scan off
exit
```

### En ambos casos

Repite con el segundo sensor (si ya lo tienes) y pega una pegatina física
a cada uno indicando dónde lo vas a instalar (te hará falta ese nombre de
sitio más adelante, para `ubicacion` en la configuración).

---

## Fase 6 — Preparar la Raspberry Pi

### 6.1 — Conectarte por primera vez

Necesitas la IP de la Raspberry Pi (mírala en tu router, o con `hostname -I`
si tienes un teclado/pantalla conectados directamente a ella) y su usuario
(por defecto suele ser `pi`).

```bash
ssh pi@IP_DE_LA_RASPBERRY
```

La primera vez te pedirá la contraseña de la Pi.

### 6.2 — Copiar tu clave SSH a la Pi (para no repetir la contraseña cada vez)

Desde tu WSL (no desde dentro de la Pi):

```bash
ssh-copy-id pi@IP_DE_LA_RASPBERRY
```

### 6.3 — Instalar lo necesario dentro de la Pi

Conéctate (`ssh pi@IP_DE_LA_RASPBERRY`) y ejecuta:

```bash
sudo apt update
sudo apt install -y git golang bluez

curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
```

Cierra la sesión SSH (`exit`) y vuelve a entrar para que el grupo `docker`
tenga efecto.

### 6.4 — Clonar tu proyecto desde GitHub

Sigue conectado a la Pi por SSH. Como la Pi también necesita autenticarse
contra GitHub, genera una clave SSH **dentro de la Pi** (es distinta de la
de tu WSL) y añádela a GitHub igual que hiciste en el paso 4.3:

```bash
ssh-keygen -t ed25519 -C "raspberry-cpd@tu-empresa.com"
cat ~/.ssh/id_ed25519.pub
# pégala en GitHub → Settings → SSH and GPG keys → New SSH key

git clone git@github.com:alexio9910/cpd-monitor.git
cd cpd-monitor
```

### 6.5 — Crear el usuario de sistema que ejecutará el servicio

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin cpdmonitor
sudo usermod -aG bluetooth cpdmonitor
```

---

## Fase 7 — Levantar InfluxDB y Grafana en la Pi

Todavía dentro de la Pi, en la carpeta `cpd-monitor`:

```bash
cp .env.example .env
nano .env
```

En el editor `nano` (se navega con las flechas), rellena por ahora:
- `INFLUXDB_ADMIN_PASSWORD`: una contraseña que te inventes.
- `INFLUXDB_ADMIN_TOKEN`: genera uno con `openssl rand -hex 32` en otra
  terminal y pégalo aquí.
- `GRAFANA_ADMIN_PASSWORD`: otra contraseña que te inventes.

Deja `GRAFANA_INFLUXDB_TOKEN` y `GRAFANA_VIEWER_PASSWORD` en blanco — se
rellenan solos en el paso 7.2, porque necesitan datos que todavía no
existen. Si no vas a configurar alertas por email ahora mismo, deja
también en blanco `GRAFANA_SMTP_*`, `GRAFANA_ALERT_EMAILS` y
`GRAFANA_ORG_NAME` — Grafana arranca igual sin ellas (ver Fase 10.7 para
configurarlas más adelante).

Guarda con `Ctrl+O`, `Enter`, y sal con `Ctrl+X`.

### 7.1 — Levantar los contenedores

```bash
docker compose up -d
docker compose ps
```

Deberías ver `cpd-influxdb` y `cpd-grafana` con estado "running"/"Up".

### 7.2 — Crear un token de InfluxDB de solo lectura para Grafana (obligatorio)

Grafana necesita un token para leer los datos de InfluxDB. **No uses el
token de administrador para esto** — le daría permisos de sobra (borrar
buckets, crear usuarios...) que no necesita solo para dibujar gráficas.
Creamos uno aparte, limitado a **solo lectura** del bucket de este
proyecto:

```bash
ADMIN_TOKEN=$(grep '^INFLUXDB_ADMIN_TOKEN=' .env | cut -d'=' -f2)
ORG=$(grep '^INFLUXDB_ORG=' .env | cut -d'=' -f2)
BUCKET=$(grep '^INFLUXDB_BUCKET=' .env | cut -d'=' -f2)

BUCKET_ID=$(docker exec cpd-influxdb influx bucket list --org "$ORG" --token "$ADMIN_TOKEN" --hide-headers | awk -v b="$BUCKET" '$2==b {print $1}')

GRAFANA_TOKEN=$(docker exec cpd-influxdb influx auth create \
  --org "$ORG" --token "$ADMIN_TOKEN" \
  --read-bucket "$BUCKET_ID" \
  --description "grafana-solo-lectura" \
  --json | python3 -c "import json,sys; print(json.load(sys.stdin)['token'])")

echo "GRAFANA_INFLUXDB_TOKEN=${GRAFANA_TOKEN}" >> .env
echo "Token creado y guardado en .env"
```

Y recreamos Grafana para que lo recoja:

```bash
docker compose up -d --force-recreate grafana
docker compose ps
```

> ⚠️ Sin este paso, Grafana arranca sin dar ningún error, pero **el
> dashboard no mostrará ningún dato**. Es la causa más común de
> "Grafana está vacío" en una instalación nueva — si te pasa, revisa que
> este paso se ejecutó bien.

---

## Fase 8 — Configurar y compilar el colector

```bash
cp config.example.yaml config.yaml
nano config.yaml
```

Rellena:
- `influxdb.token`: el mismo valor que pusiste en `INFLUXDB_ADMIN_TOKEN`.
- `influxdb.org` / `influxdb.bucket`: los mismos valores que en `.env`
  (`mi-empresa` / `cpd_monitorizacion` si no los cambiaste).
- `sensores`: sustituye las MAC de ejemplo por las que anotaste en la
  Fase 5, y `ubicacion` por dónde está cada uno.

Guarda (`Ctrl+O`, `Enter`, `Ctrl+X`) y compila:

```bash
make build
sudo ./bin/cpd-monitor -config config.yaml
```

Deberías ver una línea por sensor cada 60 segundos, por ejemplo:

```
2026/08/11 10:15:03 [pasillo_frio_a] 21.40°C  45.2%HR  bateria=87%
```

> 💡 El **primer ciclo** puede tardar más de lo normal o incluso fallar con
> `tiempo de espera agotado resolviendo servicios BLE`: es la primera vez
> que BlueZ conecta con el sensor en esta sesión y tiene que recorrer todo
> su árbol de servicios GATT. A partir del segundo ciclo, con la conexión
> ya establecida y mantenida (ver `docs/ARQUITECTURA.md`), las lecturas
> deberían ser rápidas y estables.

Si funciona, para el programa con `Ctrl+C` y sigue a la Fase 9. Si da un
error persistente, mira la tabla de la Fase 12 antes de seguir.

---

## Fase 9 — Ponerlo en producción (arranque automático)

```bash
sudo mkdir -p /opt/cpd-monitor
sudo cp bin/cpd-monitor config.yaml /opt/cpd-monitor/
sudo chown -R cpdmonitor:cpdmonitor /opt/cpd-monitor

sudo cp deploy/systemd/cpd-monitor.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cpd-monitor

sudo systemctl status cpd-monitor
```

Si ves "active (running)" en verde, ya está funcionando de forma permanente:
arrancará solo cada vez que encienda la Raspberry Pi, y se reiniciará solo
si el proceso llegara a fallar.

Para ver los registros en cualquier momento:

```bash
journalctl -u cpd-monitor -f
```

(sal de esta vista con `Ctrl+C`; el servicio sigue corriendo igualmente).

---

## Fase 10 — Ver los datos en Grafana

Desde cualquier navegador de tu red: `http://IP_DE_LA_RASPBERRY:3000`.
Entra con el usuario/contraseña que pusiste en `.env` y abre el dashboard
**"CPD - Temperatura y Humedad"**. En un par de minutos deberías ver ambos
sensores con datos.

---

## Fase 10.5 — Añadir un sensor adicional (cuando llegue)

Cuando tengas el segundo sensor físico, **no hace falta recompilar nada**:
solo editar un fichero de configuración y reiniciar el servicio. Sigue
conectado a la Raspberry Pi por SSH.

### 1. Identifica la MAC del nuevo sensor

Con el sensor encendido y cerca de la Pi:

```bash
bluetoothctl
scan on
```

Espera a ver una línea con su nombre (algo como `SHT40 Gadget`) y anota su
MAC. Luego:

```
scan off
exit
```

(Es el mismo procedimiento de la Fase 5/sección 1 del README — lo tienes
más detallado ahí si lo necesitas.)

### 2. Edita el fichero de configuración de trabajo

```bash
cd ~/cpd-monitor
nano config.yaml
```

Añade un bloque nuevo bajo `sensores:`, con un `id` distinto al que ya
tienes (por ejemplo `sensor2`) y la MAC que acabas de anotar:

```yaml
sensores:
  - id: "sensor1"
    mac: "E4:E9:00:CB:E2:DC"
    ubicacion: "Sensor 1"
  - id: "sensor2"
    mac: "AA:BB:CC:DD:EE:FF"
    ubicacion: "Sensor 2"
```

Guarda con `Ctrl+O`, `Enter`, y sal con `Ctrl+X`.

### 3. Despliega el cambio y reinicia el servicio

El servicio en producción lee su propia copia de `config.yaml`, guardada en
`/opt/cpd-monitor/`, que es independiente de la que acabas de editar en
`~/cpd-monitor/`. Hay que copiar la nueva versión ahí y reiniciar:

```bash
sudo cp config.yaml /opt/cpd-monitor/config.yaml
sudo chown cpdmonitor:cpdmonitor /opt/cpd-monitor/config.yaml
sudo systemctl restart cpd-monitor
```

### 4. Comprueba que ambos sensores están leyendo

```bash
journalctl -u cpd-monitor -f
```

Deberías ver una línea por cada sensor en cada ciclo de 60 segundos. Es
normal que el sensor recién añadido tarde un poco más en su primerísimo
ciclo (BlueZ tiene que descubrirlo y resolver su árbol de servicios GATT
por primera vez), igual que pasó con el primero.

Sal con `Ctrl+C` cuando lo confirmes (el servicio sigue corriendo igual).

### 5. Añade los paneles del sensor nuevo en Grafana

Las dos gráficas grandes (Temperatura, Humedad) agrupan las series por
`sensor_id`/`ubicacion`, así que el sensor nuevo aparece solo en ellas,
como una línea adicional — sin tocar nada.

Los tres paneles pequeños de "valor actual" (Temperatura, Humedad,
Batería) sí hay que crearlos para el sensor nuevo — con un script ya
preparado para ello, sin tocar la interfaz de Grafana a mano:

```bash
cd ~/cpd-monitor
python3 scripts/anadir_sensor_dashboard.py sensor2 "Sensor 2"
docker compose restart grafana
```

El primer argumento es el mismo `id` que pusiste en `config.yaml`; el
segundo, el texto que quieres que aparezca en el título de los paneles.
Sirve igual para un tercer sensor, un cuarto, etc. — y es seguro
ejecutarlo dos veces por error: si detecta que ya existen los paneles de
ese sensor, no hace nada.

> 💡 `config.yaml` contiene tu token real de InfluxDB y está en
> `.gitignore` — nunca se sube a GitHub. Añadir un sensor así es un cambio
> puramente local en el host, sin necesidad de hacer commit ni push.

---

## Fase 10.6 — Usuario de solo visualización en Grafana (recomendado)

Para dar acceso a alguien que solo necesite **ver** el dashboard, sin
poder editar ni tocar configuración (por ejemplo, un cliente o un
compañero), crea un segundo usuario en Grafana con rol *Viewer*:

```bash
cd ~/cpd-monitor
GRAFANA_ADMIN_USER=$(grep '^GRAFANA_ADMIN_USER=' .env | cut -d'=' -f2)
GRAFANA_ADMIN_PASSWORD=$(grep '^GRAFANA_ADMIN_PASSWORD=' .env | cut -d'=' -f2)
VIEWER_PASSWORD=$(openssl rand -base64 18)

RESPUESTA=$(curl -s -u "${GRAFANA_ADMIN_USER}:${GRAFANA_ADMIN_PASSWORD}" \
  -X POST http://localhost:3000/api/admin/users \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Visor CPD\",\"login\":\"visor\",\"password\":\"${VIEWER_PASSWORD}\",\"email\":\"visor@cpd.local\"}")

VISOR_ID=$(echo "$RESPUESTA" | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")

curl -s -u "${GRAFANA_ADMIN_USER}:${GRAFANA_ADMIN_PASSWORD}" \
  -X PATCH "http://localhost:3000/api/org/users/${VISOR_ID}" \
  -H "Content-Type: application/json" \
  -d '{"role":"Viewer"}'

echo "GRAFANA_VIEWER_PASSWORD=${VIEWER_PASSWORD}" >> .env
echo
echo "Usuario: visor | Contraseña: ${VIEWER_PASSWORD}"
```

Usuario `visor`, contraseña guardada también en `.env`. Pruébalo en una
ventana de incógnito: debería poder ver el dashboard pero no editar nada.

---

## Fase 10.7 — Alertas por email (recomendado)

Que Grafana te avise por email cuando la temperatura o humedad se salgan
de rango normal. Necesitas los datos de un servidor SMTP (de tu empresa,
o un proveedor como Gmail con contraseña de aplicación).

### 10.7.1 — Configura el SMTP y los destinatarios

```bash
nano .env
```

Rellena (si no lo hiciste ya en la Fase 7):
- `GRAFANA_SMTP_HOST`: servidor y puerto, ej. `smtp.tuempresa.com:587`
- `GRAFANA_SMTP_USER` / `GRAFANA_SMTP_PASSWORD`: credenciales de esa
  cuenta de correo
- `GRAFANA_SMTP_FROM_ADDRESS`: dirección que aparecerá como remitente
- `GRAFANA_ALERT_EMAILS`: a quién avisar, varias direcciones separadas
  por `;` (ej. `"persona1@empresa.com;persona2@empresa.com"`)
- `GRAFANA_PUBLIC_URL`: la URL desde la que se accede al dashboard (ej.
  `http://IP_DE_TU_RASPBERRY:3000`) — se usa como enlace dentro del email
- `GRAFANA_ORG_NAME`: el nombre que quieras que aparezca al pie del email

Guarda y aplica:

```bash
docker compose up -d --force-recreate grafana
```

En Grafana → **Alerting → Contact points → `email-cpd` → Test**, y
confirma que te llega un correo de prueba a las direcciones que pusiste.

### 10.7.2 — Crea las dos reglas de alerta

Desde la interfaz (no por fichero, para poder revisarlas con calma):
**Alerting → Alert rules → New alert rule**. Repite dos veces, con estos
valores:

| | Temperatura | Humedad |
|---|---|---|
| Nombre | Temperatura CPD fuera de rango | Humedad CPD fuera de rango |
| Datasource / query | `InfluxDB-CPD`, campo `temperatura_c` | `InfluxDB-CPD`, campo `humedad_rel_pct` |
| Condición (Threshold) | IS BELOW 18 OR IS ABOVE 27 | IS BELOW 40 OR IS ABOVE 60 |

Para ambas, en **"Definir comportamiento de evaluación"**:
- Grupo de evaluación: crea uno nuevo, ej. "Monitorizacion-CPD", cada
  **10s** (el mínimo que permite Grafana).
- **Periodo pendiente**: `Ninguno` (avisa al instante, sin esperar
  confirmación de un segundo ciclo).
- **Seguir activando durante**: `Ninguno` (el email de "resuelto" llega
  también al instante en cuanto vuelve a rango).
- **Estado si no hay datos**: `Alerting` (para que un colector caído
  también dispare un aviso).
- **Estado en caso de error**: `Alerting` (igual, para fallos de consulta
  a InfluxDB).

Guarda cada regla con el botón final **"Save rule and exit"** — es fácil
quedarse a medias si solo se cambian los campos sin pulsar este botón.

Consulta Flux de referencia para ambas reglas (cambia solo el `_field`):

```
from(bucket: "cpd_monitorizacion")
  |> range(start: -10m)
  |> filter(fn: (r) => r._measurement == "cpd_ambiente")
  |> filter(fn: (r) => r._field == "temperatura_c")
  |> last()
  |> keep(columns: ["_time", "_value", "sensor_id", "ubicacion"])
```

> 📄 Para cambiar quién recibe las alertas más adelante, o si algo no
> llega, consulta `docs/MANUAL-EXPERTO.md` (secciones 3 y 4).

---

## Fase 10.8 — IP fija para la Raspberry Pi (recomendado en producción)

Por defecto, la IP de la Pi puede cambiar con el tiempo (reinicio del
router, corte de luz...) — si cambia, dejan de funcionar el enlace de los
emails de alerta, tus accesos SSH guardados, y `deploy.sh`. Antes de dar
el proyecto por definitivo, fíjala.

Los pasos completos (dos opciones: reserva en el router, o `nmcli`
directamente en la Pi) están en `docs/MANUAL-EXPERTO.md`, sección 11 —
para no duplicar la misma explicación en dos sitios distintos y que se
desincronicen con el tiempo.

---

## Fase 11 — Tu flujo de trabajo a partir de ahora

Cada vez que quieras cambiar algo del proyecto (por ejemplo, tocar el
dashboard o añadir un tercer sensor), hazlo **en tu WSL**, no directamente
en la Pi:

```bash
# en tu WSL, tras editar lo que sea
git add .
git commit -m "explica aquí qué has cambiado"
git push
```

Y luego, para llevarlo a producción:

```bash
# también desde tu WSL
./deploy.sh
```

Ese script se conecta por SSH a la Pi, baja los cambios (`git pull`), reconstruye lo necesario y reinicia el servicio — todo en un solo comando.

`deploy.sh` **no lleva la IP de la Pi escrita dentro** (así nunca queda
expuesta en un repositorio público) — la lee de una variable `PI_HOST` en
tu `.env` **local, de tu WSL** (uno distinto al `.env` de la Pi, que no
tiene por qué llevar las mismas variables). La primera vez:

```bash
echo "PI_HOST=tuusuario@IP_DE_LA_PI" >> .env
```

(sustituye por tu usuario e IP reales). Como `.env` está en
`.gitignore`, este paso **no requiere ningún commit**.

---

## Fase 12 — Si algo falla

| Síntoma | Qué probar |
|---|---|
| `Permission denied (publickey)` al hacer `git push` o `git clone` | Tu clave SSH no está añadida en GitHub, o le falta añadirla en esa máquina concreta (WSL y Pi tienen claves distintas). Repite el paso 4.3 o 6.4. |
| `docker: command not found` | Cierra y abre la terminal tras instalar Docker; si sigue sin ir, reinstala con el comando de la Fase 1/6.3. |
| `no se pudo conectar con el sensor ...` | El sensor está apagado, fuera de alcance, o con la pila agotada. Comprueba con `bluetoothctl scan on`. |
| `tiempo de espera agotado resolviendo servicios BLE del sensor ...` | Normal en el primer ciclo tras un reinicio del host (BlueZ redescubre el sensor desde cero). Si persiste en ciclos posteriores, sube `timeout_conexion_segundos` en `config.yaml`. |
| `Method "Get"/"Connect" ... doesn't exist` (error de D-Bus) | Bug de compatibilidad conocido de BlueZ reciente con ciertas librerías BLE — por eso el colector habla con BlueZ directamente (ver `docs/ARQUITECTURA.md`). Si aparece, `sudo systemctl restart bluetooth` y deja que el siguiente ciclo redescubra el sensor. |
| El colector no arranca como servicio pero sí a mano con `sudo` | El usuario `cpdmonitor` no está en el grupo `bluetooth` (Fase 6.5), o hace falta `sudo systemctl daemon-reload` tras copiar el `.service`. |
| Grafana no muestra datos pero el colector no da error | Revisa que `influxdb.org`/`influxdb.bucket` en `config.yaml` coincidan exactamente con `INFLUXDB_ORG`/`INFLUXDB_BUCKET` de `.env`. |
| No sabes qué significa un mensaje de error | Cópialo tal cual (todo el texto) y pégamelo — dime también en qué fase estabas. |

---

## Fase 13 — Seguir con Claude Desktop / Claude Code

Todo lo anterior lo puedes teclear tú a mano siguiendo esta guía, o puedes
abrir **Claude Desktop**, entrar en la pestaña de **Code**, apuntarlo a tu
carpeta `~/proyectos/cpd-monitor` de WSL, y pegarle esta misma guía
(o simplemente decirle "sigamos la guía desde la Fase 1"). Como esa sesión
sí tiene acceso real a tu terminal, puede ejecutar los comandos por ti,
enseñarte el resultado antes de continuar, y corregir sobre la marcha si
algo no sale como se espera — algo que yo, aquí en el chat, no puedo hacer
porque no tengo acceso a tu WSL ni a tu Raspberry Pi real.
