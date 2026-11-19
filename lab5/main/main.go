package main

import (
	"flag"
	"fmt"
	"time"
)

const assignedBits = 8 * 1024 * 1024 // 8 Мбит = 8 388 608 однобитовых ячеек

type FaultKind string

const (
	NoFault            FaultKind = "NONE"
	AFAddressNoCell    FaultKind = "AF_ADDRESS_NO_CELL"
	AFCellInaccessible FaultKind = "AF_CELL_INACCESSIBLE"
	AFMultiSelect      FaultKind = "AF_MULTI_SELECT"
	AFAlias            FaultKind = "AF_ALIAS"
	ReadStuck0         FaultKind = "READ_STUCK_0"
	ReadStuck1         FaultKind = "READ_STUCK_1"
	WriteInhibit       FaultKind = "WRITE_INHIBIT"
	WriteInvert        FaultKind = "WRITE_INVERT"
	StuckAt0           FaultKind = "SAF0"
	StuckAt1           FaultKind = "SAF1"
	TransitionUp       FaultKind = "TF_UP"
	TransitionDown     FaultKind = "TF_DOWN"
	StuckOpen          FaultKind = "SOF"
	ReadDisturb        FaultKind = "RDF"
	WeakCell           FaultKind = "WCF"
	FalseWriteThrough  FaultKind = "FWTF"
	InversionCF        FaultKind = "CFin"
	IdempotentCF       FaultKind = "CFid"
)

type Fault struct {
	Kind              FaultKind
	Aggressor, Victim int
	Direction         int // CFin/CFid: 0 = 0->1, 1 = 1->0
	Order             int // -1 = aggressor < victim, +1 = aggressor > victim
	VictimValue       uint8
	DecayAfter        uint64
}

type Memory struct {
	cells        []uint8
	fault        Fault
	lastBus      uint8
	operations   uint64
	weakDeadline uint64
	weakArmed    bool
}

func NewMemory(n int, f Fault) *Memory {
	if f.DecayAfter == 0 {
		f.DecayAfter = 8
	}
	return &Memory{cells: make([]uint8, n), fault: f}
}
func (m *Memory) valid(a int) bool { return a >= 0 && a < len(m.cells) }
func (m *Memory) tick() {
	m.operations++
	if m.fault.Kind == WeakCell && m.weakArmed && m.operations >= m.weakDeadline {
		m.cells[m.fault.Aggressor] = 0
		m.weakArmed = false
	}
}

func (m *Memory) selected(addr int) []int {
	if !m.valid(addr) {
		return nil
	}
	switch m.fault.Kind {
	case AFAddressNoCell, AFCellInaccessible:
		if addr == m.fault.Aggressor {
			return nil
		}
	case AFAlias:
		if addr == m.fault.Aggressor {
			return []int{m.fault.Victim}
		}
	case AFMultiSelect:
		if addr == m.fault.Aggressor {
			return []int{m.fault.Aggressor, m.fault.Victim}
		}
	}
	return []int{addr}
}

func (m *Memory) Read(addr int) uint8 {
	m.tick()
	sel := m.selected(addr)
	if len(sel) == 0 {
		return 0
	}
	p, value := sel[0], m.cells[sel[0]]
	switch m.fault.Kind {
	case ReadStuck0:
		if addr == m.fault.Aggressor {
			value = 0
		}
	case ReadStuck1:
		if addr == m.fault.Aggressor {
			value = 1
		}
	case StuckOpen:
		if addr == m.fault.Aggressor {
			value = m.lastBus
		}
	case ReadDisturb:
		if addr == m.fault.Aggressor {
			m.cells[p] ^= 1
		}
	}
	m.lastBus = value
	return value & 1
}

func (m *Memory) Write(addr int, value uint8) {
	m.tick()
	value &= 1
	sel := m.selected(addr)
	if len(sel) == 0 {
		return
	}
	if addr == m.fault.Aggressor {
		switch m.fault.Kind {
		case WriteInhibit:
			return
		case WriteInvert:
			value ^= 1
		}
	}
	for _, p := range sel {
		old := m.cells[p]
		switch m.fault.Kind {
		case StuckAt0:
			if p == m.fault.Aggressor {
				m.cells[p] = 0
				continue
			}
		case StuckAt1:
			if p == m.fault.Aggressor {
				m.cells[p] = 1
				continue
			}
		case TransitionUp:
			if p == m.fault.Aggressor && old == 0 && value == 1 {
				continue
			}
		case TransitionDown:
			if p == m.fault.Aggressor && old == 1 && value == 0 {
				continue
			}
		case StuckOpen:
			if p == m.fault.Aggressor {
				continue
			}
		}
		m.cells[p] = value
		if m.fault.Kind == WeakCell && p == m.fault.Aggressor && value == 1 {
			m.weakDeadline, m.weakArmed = m.operations+m.fault.DecayAfter, true
		}
		if p == m.fault.Aggressor && old != value && m.valid(m.fault.Victim) {
			orderOK := (m.fault.Order < 0 && m.fault.Aggressor < m.fault.Victim) ||
				(m.fault.Order > 0 && m.fault.Aggressor > m.fault.Victim)
			if orderOK {
				switch m.fault.Kind {
				case InversionCF:
					if (m.fault.Direction == 0 && old == 0 && value == 1) ||
						(m.fault.Direction == 1 && old == 1 && value == 0) {
						m.cells[m.fault.Victim] ^= 1
					}
				case IdempotentCF:
					if (m.fault.Direction == 0 && old == 0 && value == 1) ||
						(m.fault.Direction == 1 && old == 1 && value == 0) {
						m.cells[m.fault.Victim] = m.fault.VictimValue & 1
					}
				}
			}
		}
		if m.fault.Kind == FalseWriteThrough && p == m.fault.Aggressor && m.valid(m.fault.Victim) {
			m.cells[m.fault.Victim] = value
		}
	}
}

// TestResult — результат одного теста + логи ошибок
type TestResult struct {
	Name       string
	Detected   bool
	Operations uint64
	Errors     int
	Logs       []string
}

type Tester struct {
	mem  *Memory
	ops  uint64
	errs int
	logs []string
	fon  string // для лога
}

func (t *Tester) write(a int, v uint8) { t.mem.Write(a, v); t.ops++ }
func (t *Tester) read(a int, expected uint8) {
	got := t.mem.Read(a)
	t.ops++
	if got != expected {
		t.errs++
		if len(t.logs) < 5 { // ограничиваем вывод первыми 5 ошибками
			t.logs = append(t.logs, fmt.Sprintf("Фон: %-3s | Ожидаем: %d | Получаем: %d (Адрес: %d)", t.fon, expected, got, a))
		}
	}
}
func (t *Tester) result(name string) TestResult {
	return TestResult{name, t.errs > 0, t.ops, t.errs, t.logs}
}

func RunGALPAT(n int, f Fault) TestResult {
	m := NewMemory(n, f)
	t := &Tester{mem: m, fon: "0"}
	for i := 0; i < n; i++ {
		t.write(i, 0)
	}
	for base := 0; base < n; base++ {
		t.write(base, 1)
		for a := 0; a < n; a++ {
			if a != base {
				t.read(a, 0)
				t.read(base, 1)
			}
		}
		t.write(base, 0)
	}
	return t.result("GALPAT")
}

func RunGALPATComplement(n int, f Fault) TestResult {
	m := NewMemory(n, f)
	t := &Tester{mem: m, fon: "1"}
	for i := 0; i < n; i++ {
		t.write(i, 1)
	}
	if m.valid(f.Victim) {
		t.write(f.Victim, 1)
	}
	for base := 0; base < n; base++ {
		t.write(base, 0)
		for a := 0; a < n; a++ {
			if a != base {
				t.read(a, 1)
				t.read(base, 0)
			}
		}
		t.write(base, 1)
	}
	return t.result("GALPAT (фон 1)")
}

func RunMarchA(n int, f Fault) TestResult {
	m := NewMemory(n, f)
	t := &Tester{mem: m, fon: "0/1"}
	for i := 0; i < n; i++ {
		t.write(i, 0)
	}
	for i := 0; i < n; i++ {
		t.read(i, 0)
		t.write(i, 1)
		t.write(i, 0)
		t.write(i, 1)
	}
	for i := n - 1; i >= 0; i-- {
		t.read(i, 1)
		t.write(i, 0)
		t.write(i, 1)
	}
	for i := 0; i < n; i++ {
		t.read(i, 1)
		t.write(i, 0)
		t.write(i, 1)
		t.write(i, 0)
	}
	for i := n - 1; i >= 0; i-- {
		t.read(i, 0)
		t.write(i, 1)
		t.write(i, 0)
	}
	return t.result("March A")
}

type FaultCase struct {
	Label string
	Fault Fault
}

func faultCases(n int) []FaultCase {
	a, v := n/3, 2*n/3
	if a == v {
		v = (v + 1) % n
	}
	cases := []FaultCase{
		{"AF: заданный адрес не выбирает ни одной ячейки", Fault{Kind: AFAddressNoCell, Aggressor: a}},
		{"AF: заданная ячейка недоступна", Fault{Kind: AFCellInaccessible, Aggressor: a}},
		{"AF: адрес выбирает несколько ячеек", Fault{Kind: AFMultiSelect, Aggressor: a, Victim: v}},
		{"AF: ячейка доступна по нескольким адресам", Fault{Kind: AFAlias, Aggressor: a, Victim: v}},
		{"Логика чтения: принудительный 0", Fault{Kind: ReadStuck0, Aggressor: a}},
		{"Логика чтения: принудительная 1", Fault{Kind: ReadStuck1, Aggressor: a}},
		{"Логика записи: запись запрещена", Fault{Kind: WriteInhibit, Aggressor: a}},
		{"Логика записи: инверсия данных", Fault{Kind: WriteInvert, Aggressor: a}},
		{"SAF0: константный ноль", Fault{Kind: StuckAt0, Aggressor: a}},
		{"SAF1: константная единица", Fault{Kind: StuckAt1, Aggressor: a}},
		{"TF: переход 0->1 запрещен", Fault{Kind: TransitionUp, Aggressor: a}},
		{"TF: переход 1->0 запрещен", Fault{Kind: TransitionDown, Aggressor: a}},
		{"SOF: обрыв (удержание предыдущего значения шины)", Fault{Kind: StuckOpen, Aggressor: a}},
		{"RDF: чтение изменяет содержимое", Fault{Kind: ReadDisturb, Aggressor: a}},
		{"WCF: слабое хранение единицы", Fault{Kind: WeakCell, Aggressor: a, DecayAfter: 8}},
		{"FWTF: запись проходит в постороннюю ячейку", Fault{Kind: FalseWriteThrough, Aggressor: a, Victim: v}},
	}
	for _, order := range []int{-1, 1} {
		for dir := 0; dir <= 1; dir++ {
			ag, vi := a, v
			if order > 0 {
				ag, vi = v, a
			}
			cases = append(cases, FaultCase{
				fmt.Sprintf("CFin: %s; %s", map[int]string{0: "0->1", 1: "1->0"}[dir], map[int]string{-1: "agg<vic", 1: "agg>vic"}[order]),
				Fault{Kind: InversionCF, Aggressor: ag, Victim: vi, Direction: dir, Order: order},
			})
		}
	}
	for _, order := range []int{-1, 1} {
		for dir := 0; dir <= 1; dir++ {
			for vv := uint8(0); vv <= 1; vv++ {
				ag, vi := a, v
				if order > 0 {
					ag, vi = v, a
				}
				// Названия исправлены: "принудительный" вместо "жертва"
				cases = append(cases, FaultCase{
					fmt.Sprintf("CFid: %s; принудительный %d; %s",
						map[int]string{0: "0->1", 1: "1->0"}[dir], vv,
						map[int]string{-1: "agg<vic", 1: "agg>vic"}[order]),
					Fault{Kind: IdempotentCF, Aggressor: ag, Victim: vi, Direction: dir, Order: order, VictimValue: vv},
				})
			}
		}
	}
	return cases
}

// printResult — печать результата теста + логов
func printResult(r TestResult) {
	status := "НЕ ОБНАРУЖЕНА"
	if r.Detected {
		status = "ОБНАРУЖЕНА"
	}
	fmt.Printf("  %-15s | операций: %-6d | ошибок: %-4d | %s\n", r.Name, r.Operations, r.Errors, status)
	if r.Detected && len(r.Logs) > 0 {
		for _, log := range r.Logs {
			fmt.Printf("    -> %s\n", log)
		}
		if r.Errors > len(r.Logs) {
			fmt.Printf("    -> ... (и ещё %d ошибок)\n", r.Errors-len(r.Logs))
		}
	}
}

// Таблица покрытия
type coverageRow struct {
	Label      string
	GALPAT     bool
	GALPATComp bool
	MarchA     bool
}

func main() {
	n := flag.Int("n", 256, "число моделируемых однобитовых ячеек")
	flag.Parse()
	if *n < 2 {
		fmt.Println("Размер модели должен быть не менее 2")
		return
	}

	fmt.Println("Лабораторная работа №7. Вариант 9")
	fmt.Printf("Объем ОЗУ по заданию: 8 Мбит = %d бит; размер модели: %d ячеек\n\n", assignedBits, *n)
	fmt.Println("Для каждой неисправности выполняются GALPAT, GALPAT (фон 1) и March A.")
	fmt.Println("При обнаружении ошибок выводятся данные: (Фон | Ожидаем | Получаем)\n")

	var coverage []coverageRow

	for i, fc := range faultCases(*n) {
		fmt.Printf("[%02d] %s\n", i+1, fc.Label)
		start := time.Now()

		g := RunGALPAT(*n, fc.Fault)
		gc := RunGALPATComplement(*n, fc.Fault)
		ma := RunMarchA(*n, fc.Fault)

		printResult(g)
		printResult(gc)
		printResult(ma)

		fmt.Printf("  Время: %s\n\n", time.Since(start))
		coverage = append(coverage, coverageRow{fc.Label, g.Detected, gc.Detected, ma.Detected})
	}

	// ===== ИТОГОВАЯ СВОДНАЯ ТАБЛИЦА =====
	fmt.Println("\n================ СВОДНАЯ ТАБЛИЦА ПОКРЫТИЯ ================")
	fmt.Printf("%-52s | GAL | GAL1 | March\n", "Сценарий")
	fmt.Println("------------------------------------------------------------")
	totalG, totalGC, totalMA := 0, 0, 0
	for _, r := range coverage {
		mark := func(b bool) string {
			if b {
				return " +  "
			}
			return " -  "
		}
		fmt.Printf("%-52s | %s | %s | %s\n", r.Label, mark(r.GALPAT), mark(r.GALPATComp), mark(r.MarchA))
		if r.GALPAT {
			totalG++
		}
		if r.GALPATComp {
			totalGC++
		}
		if r.MarchA {
			totalMA++
		}
	}
	fmt.Println("------------------------------------------------------------")
	total := len(coverage)
	fmt.Printf("Обнаружено: GALPAT %d/%d, GALPAT(фон1) %d/%d, March A %d/%d\n", totalG, total, totalGC, total, totalMA, total)

	// Оценка сложности
	N := uint64(assignedBits)
	fmt.Println("\nОценка сложности для полного массива:")
	fmt.Printf("GALPAT: 2N^2+6N = %d операций\n", 2*N*N+6*N)
	fmt.Printf("March A: 15N = %d операций\n", 15*N)
	fmt.Println("Полный GALPAT практически невыполним; эксперимент на уменьшенном размере -n.")
}
