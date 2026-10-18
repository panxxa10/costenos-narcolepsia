package hospital

import "fmt"

// Person agrupa los datos básicos de cualquier persona del hospital.
// Se embebe por valor en Patient, Doctor y Orderly para no repetir campos.
type Person struct {
	id   string
	name string
	age  int
}

// newPerson construye una Person. Es interna porque desde fuera del paquete
// las personas se crean con NewPatient, NewDoctor o NewOrderly.
func newPerson(id, name string, age int) Person {
	return Person{id: id, name: name, age: age}
}

// ID devuelve el identificador único de la persona.
func (p Person) ID() string { return p.id }

// Name devuelve el nombre completo de la persona.
func (p Person) Name() string { return p.name }

// Age devuelve la edad en años.
func (p Person) Age() int { return p.age }

// NarcolepsyLevel indica qué tan fuerte es la narcolepsia del paciente.
type NarcolepsyLevel int

// Niveles de narcolepsia.
const (
	Mild NarcolepsyLevel = iota
	Moderate
	Severe
)

// String devuelve el nivel en texto legible.
func (l NarcolepsyLevel) String() string {
	switch l {
	case Mild:
		return "leve"
	case Moderate:
		return "moderada"
	case Severe:
		return "severa"
	default:
		return fmt.Sprintf("NarcolepsyLevel(%d)", int(l))
	}
}

// PatientState indica si el paciente está despierto o dónde está dormido.
type PatientState int

// Estados posibles de un paciente.
const (
	Awake PatientState = iota
	AsleepInHallway
	AsleepInBed
)

// String devuelve el estado en texto legible.
func (s PatientState) String() string {
	switch s {
	case Awake:
		return "despierto"
	case AsleepInHallway:
		return "dormido en el pasillo"
	case AsleepInBed:
		return "dormido en cama"
	default:
		return fmt.Sprintf("PatientState(%d)", int(s))
	}
}

// Patient es un paciente costeño con narcolepsia.
// No es seguro para uso concurrente por sí solo: desde varias goroutines
// se modifica únicamente a través de los métodos del Hospital.
type Patient struct {
	Person
	level           NarcolepsyLevel
	state           PatientState
	currentLocation string
	assignedDoctor  *Doctor
	room            *Room
}

// NewPatient crea un paciente despierto en la consulta externa.
func NewPatient(id, name string, age int, level NarcolepsyLevel) *Patient {
	return &Patient{
		Person:          newPerson(id, name, age),
		level:           level,
		state:           Awake,
		currentLocation: "consulta externa",
	}
}

// Level devuelve el nivel de narcolepsia del paciente.
func (p *Patient) Level() NarcolepsyLevel { return p.level }

// State devuelve el estado actual del paciente.
func (p *Patient) State() PatientState { return p.state }

// Location devuelve dónde está el paciente ahora mismo.
func (p *Patient) Location() string { return p.currentLocation }

// AssignedDoctor devuelve el doctor a cargo, o nil si todavía no tiene.
func (p *Patient) AssignedDoctor() *Doctor { return p.assignedDoctor }

// Room devuelve la sala donde duerme el paciente, o nil si no tiene.
func (p *Patient) Room() *Room { return p.room }

// SufferSleepAttack hace que el paciente se duerma de repente en location.
// Queda tirado en el pasillo hasta que alguien le consiga cama.
func (p *Patient) SufferSleepAttack(location string) {
	p.state = AsleepInHallway
	p.currentLocation = location
}

// WakeUp despierta al paciente y, si estaba en una sala, la libera.
// Despierto, vuelve a la consulta externa.
// Devuelve ErrAlreadyAwake si ya estaba despierto.
func (p *Patient) WakeUp() error {
	if p.state == Awake {
		return fmt.Errorf("%s: %w", p.Name(), ErrAlreadyAwake)
	}
	if p.room != nil {
		if err := p.room.Release(p); err != nil {
			return err
		}
		p.room = nil
	}
	p.state = Awake
	p.currentLocation = "consulta externa"
	return nil
}

// goToBed acuesta al paciente en la sala r. Solo la usa el Hospital
// después de que r.Occupy(p) haya funcionado.
func (p *Patient) goToBed(r *Room) {
	p.state = AsleepInBed
	p.room = r
	p.currentLocation = fmt.Sprintf("sala %d", r.Number())
}
