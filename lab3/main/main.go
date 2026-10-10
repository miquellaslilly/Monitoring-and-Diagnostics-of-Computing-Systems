package main

import (
	"fmt"
)

// ==================== Параметры варианта ====================

const (
	VariantNumber = 9
	PolyString    = "x^8 ⊕ x^5 ⊕ x^3 ⊕ x ⊕ 1"
)

// ==================== Модель LFSR ====================

type LFSR struct {
	state uint8
	taps  []int
	deg   int
}

func NewLFSR(taps []int, deg int, seed uint8) *LFSR {
	return &LFSR{state: seed, taps: taps, deg: deg}
}

func (l *LFSR) Next() uint8 {
	fb := 0
	for _, t := range l.taps {
		fb ^= int((l.state >> uint(t)) & 1)
	}
	l.state = (l.state >> 1) | uint8(fb<<uint(l.deg-1))
	return l.state
}

func (l *LFSR) State() uint8     { return l.state }
func (l *LFSR) Reset(seed uint8) { l.state = seed }

// ==================== Модель неисправностей ====================

type Fault struct {
	Element string
	Pin     string
	Value   bool
}

func (f Fault) String() string {
	val := 0
	if f.Value {
		val = 1
	}
	return fmt.Sprintf("%s.%s=SA%d", f.Element, f.Pin, val)
}

func allFaults() []Fault {
	var faults []Fault
	add := func(el string, pins ...string) {
		for _, p := range pins {
			faults = append(faults, Fault{el, p, false})
			faults = append(faults, Fault{el, p, true})
		}
	}
	add("F1", "in1", "in2", "out")
	add("F2", "in1", "out")
	add("F3", "in1", "in2", "in3", "out")
	add("F4", "in1", "in2", "out")
	add("F5", "in1", "in2", "out")
	add("F6", "in1", "in2", "out")
	return faults
}

// ==================== Модель схемы ====================

func evaluate(x [8]bool) (f1, f2, f3, f4, f5, f6 bool) {
	f1 = x[1] && x[2]
	f2 = !x[3]
	f3 = x[5] || x[6] || x[7]
	f4 = f3 && x[4]
	f5 = !(f2 && f4)
	f6 = f1 && f5
	return
}

func evaluateFaulty(x [8]bool, f Fault) (f1, f2, f3, f4, f5, f6 bool) {
	applyFault := func(element, pin string, normal bool) bool {
		if f.Element == element && f.Pin == pin {
			return f.Value
		}
		return normal
	}

	in1F1 := applyFault("F1", "in1", x[1])
	in2F1 := applyFault("F1", "in2", x[2])
	f1 = applyFault("F1", "out", in1F1 && in2F1)

	in1F2 := applyFault("F2", "in1", x[3])
	f2 = applyFault("F2", "out", !in1F2)

	in1F3 := applyFault("F3", "in1", x[5])
	in2F3 := applyFault("F3", "in2", x[6])
	in3F3 := applyFault("F3", "in3", x[7])
	f3 = applyFault("F3", "out", in1F3 || in2F3 || in3F3)

	in1F4 := applyFault("F4", "in1", f3)
	in2F4 := applyFault("F4", "in2", x[4])
	f4 = applyFault("F4", "out", in1F4 && in2F4)

	in1F5 := applyFault("F5", "in1", f2)
	in2F5 := applyFault("F5", "in2", f4)
	f5 = applyFault("F5", "out", !(in1F5 && in2F5))

	in1F6 := applyFault("F6", "in1", f1)
	in2F6 := applyFault("F6", "in2", f5)
	f6 = applyFault("F6", "out", in1F6 && in2F6)

	return
}

func detectFault(x [8]bool, f Fault) bool {
	_, _, _, _, _, good := evaluate(x)
	_, _, _, _, _, bad := evaluateFaulty(x, f)
	return good != bad
}

// ==================== Преобразование состояния ====================

func stateToInputs(st uint8) [8]bool {
	var x [8]bool
	for j := 1; j <= 7; j++ {
		x[j] = ((st >> uint(j-1)) & 1) == 1
	}
	return x
}

// ==================== Поиск минимального теста ====================

type TestResult struct {
	Tacts       int
	Seed        uint8
	TestVectors []uint8
}

func findMinTest(lfsr *LFSR, maxTacts int) *TestResult {
	faults := allFaults()
	totalFaults := len(faults)

	var best *TestResult

	for seed := uint8(1); seed != 0; seed++ {
		lfsr.Reset(seed)
		covered := make([]bool, totalFaults)
		coveredCount := 0
		var vectors []uint8

		for tact := 1; tact <= maxTacts; tact++ {
			st := lfsr.State()
			vectors = append(vectors, st)
			x := stateToInputs(st)

			for k, f := range faults {
				if !covered[k] && detectFault(x, f) {
					covered[k] = true
					coveredCount++
				}
			}

			if coveredCount == totalFaults {
				if best == nil || tact < best.Tacts {
					vecCopy := make([]uint8, len(vectors))
					copy(vecCopy, vectors)
					best = &TestResult{
						Tacts:       tact,
						Seed:        seed,
						TestVectors: vecCopy,
					}
				}
				break
			}

			lfsr.Next()
		}
	}
	return best
}

// ==================== Main ====================

func main() {
	fmt.Printf("=== Лабораторная работа №4, Вариант %d ===\n", VariantNumber)
	fmt.Printf("Порождающий полином: %s\n\n", PolyString)

	// Для x^8 ⊕ x^5 ⊕ x^3 ⊕ x ⊕ 1: отводы = {5, 3, 1, 0}
	taps := []int{5, 3, 1, 0}
	deg := 8

	lfsr := NewLFSR(taps, deg, 1)

	fmt.Printf("Степень полинома: %d\n", deg)
	fmt.Printf("Отводы обратной связи: %v\n", taps)
	fmt.Printf("Период M-последовательности: %d\n\n", (1<<deg)-1)

	// Первые 20 состояний
	fmt.Println("Первые 20 состояний LFSR (seed = 00000001):")
	lfsr.Reset(1)
	for i := 0; i < 20; i++ {
		st := lfsr.State()
		x := stateToInputs(st)
		_, _, _, _, _, f6 := evaluate(x)
		// Вывод x1..x7 в правильном порядке
		fmt.Printf("  Такт %2d: LFSR=%08b  x1..x7=", i, st)
		for j := 1; j <= 7; j++ {
			if x[j] {
				fmt.Print("1")
			} else {
				fmt.Print("0")
			}
		}
		fmt.Printf("  F6=%d\n", boolToInt(f6))
		lfsr.Next()
	}
	fmt.Println()

	faults := allFaults()
	fmt.Printf("Всего неисправностей в схеме: %d\n\n", len(faults))

	fmt.Println("Поиск минимального теста (перебор 255 seed'ов)...")
	result := findMinTest(lfsr, 255)

	if result == nil {
		fmt.Println("Не удалось найти тест с 100% покрытием")
		return
	}

	fmt.Printf("\n=== РЕЗУЛЬТАТ ===\n")
	fmt.Printf("Минимальное число тактов: %d\n", result.Tacts)
	fmt.Printf("Начальное состояние LFSR (seed): %08b (0x%02X)\n\n",
		result.Seed, result.Seed)

	// Детальная трассировка с покрытием
	fmt.Println("Тестовая последовательность:")
	fmt.Println("  Такт | LFSR     | x1..x7  | F6 | +покрыто | всего")
	fmt.Println("  -----+----------+---------+----+----------+-------")

	lfsr.Reset(result.Seed)
	covered := make([]bool, len(faults))
	coveredCount := 0

	for i := 0; i < result.Tacts; i++ {
		st := lfsr.State()
		x := stateToInputs(st)
		_, _, _, _, _, f6 := evaluate(x)

		newly := 0
		for k, f := range faults {
			if !covered[k] && detectFault(x, f) {
				covered[k] = true
				coveredCount++
				newly++
			}
		}

		fmt.Printf("  %4d | %08b | ", i, st)
		for j := 1; j <= 7; j++ {
			if x[j] {
				fmt.Print("1")
			} else {
				fmt.Print("0")
			}
		}
		fmt.Printf(" | %d  | +%d       | %d/%d\n",
			boolToInt(f6), newly, coveredCount, len(faults))

		lfsr.Next()
	}

	fmt.Printf("\nИтоговое покрытие: %d/%d (%.1f%%)\n",
		coveredCount, len(faults),
		100.0*float64(coveredCount)/float64(len(faults)))

	var uncovered []Fault
	for k, c := range covered {
		if !c {
			uncovered = append(uncovered, faults[k])
		}
	}
	if len(uncovered) > 0 {
		fmt.Println("\nНепокрытые неисправности:")
		for _, f := range uncovered {
			fmt.Printf("  %s\n", f.String())
		}
	} else {
		fmt.Println("\nВсе неисправности покрыты!")
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
