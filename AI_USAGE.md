# Uso de IA en el proyecto

## Herramientas usadas y para qué

- **Claude Code (modelo Claude Opus 5.5)**, en la terminal. Generó el código de los cuatro paquetes, el frontend y estos documentos.
- No usé otras herramientas de IA ni otras fuentes: todo el proyecto se hizo con Claude Code.

## Prompts decisivos

1. **El prompt principal** (resumido): el enunciado completo del "Hospital de los Costeños con Narcolepsia" con las reglas técnicas (solo stdlib, sin `panic`, errores centinela, campos no exportados, `hospital` sin presentación), la tabla de structs y métodos, la interfaz `Attender` con despacho round-robin, las 4 consultas, el mutex con helpers `...Locked`, los dos bonus, los tests y la documentación. Pedí explícitamente que el código fuera "claro, idiomático y sencillo, sin sobreingeniería" porque tengo que defender cada línea, y que las dudas se resolvieran con la opción más simple y se anotaran en el README.
2. **La parte de concurrencia del prompt**: "los métodos públicos delegan en helpers internos sin lock (por ejemplo assignRoomLocked) y nunca llaman a otro método público que vuelva a tomar el lock". Eso definió la forma de todo `hospital.go`.

## Casos donde el código generado estuvo mal o no era idiomático, y cómo se corrigió

Estos casos pasaron de verdad en la sesión de desarrollo. No hubo errores de compilación, advertencias de `go vet`, deadlocks ni carreras de datos en el código final; lo que sí pasó fue esto:

### 1. Paciente despierto con la ubicación de una sala ajena (bug real)

Al probar la API con `curl`, el `GET /api/state` mostró a Yeimy Padilla (P-002) **despierta** pero con `"Location": "sala 102"`, mientras Eliécer Mendoza (P-004) ya estaba durmiendo en esa misma sala 102. El `WakeUp()` original liberaba la sala pero no cambiaba `currentLocation`.

Corrección (commit `3bce07e`): en `Patient.WakeUp()` se agregó `p.currentLocation = "consulta externa"`, y se extendió el test `TestWakePatientReleasesRoomAndHallwayIsRelocated` para verificar la ubicación. Ningún test lo había detectado antes porque ninguno revisaba la ubicación después de despertar.

### 2. Un commit que no compila por sí solo (mala práctica de git)

Para tener commits pequeños, el commit `bf11b01` ("Modelo: estados tipados, Person embebido...") incluye `patient.go`, `room.go` y `episode.go`, pero esos archivos usan errores (`ErrAlreadyAwake`, `ErrRoomFull`...) y la interfaz `Attender`, que están en `hospital.go` y llegaron en el commit siguiente. Si alguien hace `git checkout bf11b01`, `go build` falla. Lo correcto habría sido meter esos archivos junto con lo que necesitan, o escribir el código en el orden en que se iba a commitear. No lo reescribí para no alterar el historial; queda anotado aquí. A partir del commit siguiente todos compilan.

### 3. Comprobar que el test concurrente sirve de algo (experimento, no error)

Un test que siempre pasa no prueba nada. Para verificarlo, se quitó a propósito el `h.mu.Lock()` / `defer h.mu.Unlock()` de `AssignRoom` y se corrió `go test -race -run Concurrent ./hospital`: el detector reportó **8 `WARNING: DATA RACE`** y el test falló. Con el lock restaurado (`git checkout hospital/hospital.go`) vuelve a pasar. Esto confirma que el mutex es lo que evita que dos goroutines se queden con la misma sala.

### 4. Decisiones que corrigieron el enunciado tal como estaba escrito (diseño)

- El enunciado dice "llama a su Attend, luego intenta AssignRoom". Haciéndolo en ese orden, el episodio que el doctor guarda en su lista quedaría sin sala aunque después sí se asignara. Se cambió el orden (sala primero, luego `Attend`) y se documentó en el README.

Revisando el código, no encontré otros errores aparte de los de arriba.

## Qué aprendí

- **El mutex tiene que cubrir todo el "revisar y ocupar".** Si una goroutine pregunta si la sala está libre y otra la ocupa antes de que la primera actúe, las dos se quedan con la misma sala. Lo vi cuando le quitamos el lock a `AssignRoom`: `go test -race` reportó 8 carreras y el test concurrente falló.
- **`sync.Mutex` no es reentrante.** Si un método que ya tiene el lock llama a otro método público que lo vuelve a pedir, el programa se queda trabado para siempre. Lo probamos en una copia del proyecto: el test se quedó en `sync.(*Mutex).Lock` hasta que lo mató el timeout, y `go vet` no lo detectó. Por eso los métodos públicos toman el lock y delegan en helpers `...Locked`.
- **El detector de carreras solo encuentra las carreras que ocurren mientras corre el programa.** Por eso el test lanza 20 goroutines a la vez sobre 3 salas: hay que forzar que choquen.
- **Un receptor valor trabaja sobre una copia.** Cuando cambiamos `Room.Occupy` a receptor valor, la sala nunca pasaba a ocupada y fallaron 4 tests. Con `Hospital`, `go vet` avisó que se estaba copiando el `sync.Mutex`.
- **Los campos en minúscula son la encapsulación de Go.** Desde `main`, `p.state = ...` ni siquiera compila; la única forma de cambiar el estado es con los métodos, que respetan las reglas.
- **Envolver errores con `%w` y compararlos con `errors.Is`.** El mensaje lleva contexto ("Eliécer Mendoza (P-004) sigue dormido en...") y aun así se puede saber que es `ErrNoRoomAvailable`. Un `==` no funcionaría con el error envuelto.
- **Que los tests pasen no significa que todo esté bien.** El bug de la ubicación de Yeimy no lo detectaba ningún test; apareció mirando la respuesta real de la API con `curl`.
- **La versión de `go.mod` cambia cómo se comporta el lenguaje.** Con `go 1.21`, la variable del `for` se comparte entre iteraciones (por eso se pasa como parámetro a la goroutine) y `ServeMux` no entiende rutas como `"GET /api/state"`.
