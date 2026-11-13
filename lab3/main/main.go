package main

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// ============================================================
// Лабораторная работа №5. Вариант 9
// ============================================================

const (
	VariantNumber = 9
	PolyString    = "x^8 + x^5 + x^3 + x + 1"

	LFSRDegree = 8
	MaxTacts   = 255
)

// ============================================================
// 1. Генератор тестовых воздействий на основе LFSR
// ============================================================

// Полином x^8 + x^5 + x^3 + x + 1.
// Отводы соответствуют исходной ЛР №3: [5, 3, 1, 0].

type LFSR struct {
	state uint8
	taps  []int
	deg   int
}

func NewLFSR(taps []int, deg int, seed uint8) *LFSR {
	return &LFSR{
		state: seed,
		taps:  taps,
		deg:   deg,
	}
}

func (l *LFSR) Next() uint8 {
	fb := 0

	for _, t := range l.taps {
		fb ^= int((l.state >> uint(t)) & 1)
	}

	l.state = (l.state >> 1) |
		uint8(fb<<uint(l.deg-1))

	return l.state
}

func (l *LFSR) State() uint8 {
	return l.state
}

func (l *LFSR) Reset(seed uint8) {
	l.state = seed
}

// ============================================================
// 2. Модель одиночных константных неисправностей
// ============================================================

type Fault struct {
	Element string
	Pin     string
	Value   bool
}

func (f Fault) String() string {
	value := 0
	if f.Value {
		value = 1
	}

	return fmt.Sprintf("%s.%s=SA%d",
		f.Element, f.Pin, value)
}

// Формирование 36 неисправностей согласно ЛР №3.
// Проверяются входы и выходы логических элементов.

func allFaults() []Fault {
	var faults []Fault

	add := func(element string, pins ...string) {
		for _, pin := range pins {
			faults = append(faults,
				Fault{element, pin, false},
				Fault{element, pin, true},
			)
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

// ============================================================
// 3. Исправная тестируемая схема
// ============================================================

// F1 = AND(x1, x2)
// F2 = NOT(x3)
// F3 = OR(x5, x6, x7)
// F4 = AND(F3, x4)
// F5 = NAND(F2, F4)
// F6 = AND(F1, F5)

func evaluate(x [8]bool) (
	f1, f2, f3, f4, f5, f6 bool,
) {
	f1 = x[1] && x[2]
	f2 = !x[3]
	f3 = x[5] || x[6] || x[7]
	f4 = f3 && x[4]
	f5 = !(f2 && f4)
	f6 = f1 && f5

	return
}

// ============================================================
// 4. Моделирование неисправной схемы
// ============================================================

func evaluateFaulty(x [8]bool, f Fault) (
	f1, f2, f3, f4, f5, f6 bool,
) {
	applyFault := func(
		element, pin string,
		normal bool,
	) bool {
		if f.Element == element && f.Pin == pin {
			return f.Value
		}
		return normal
	}

	// F1 = AND
	in1F1 := applyFault("F1", "in1", x[1])
	in2F1 := applyFault("F1", "in2", x[2])
	f1 = applyFault("F1", "out", in1F1 && in2F1)

	// F2 = NOT
	in1F2 := applyFault("F2", "in1", x[3])
	f2 = applyFault("F2", "out", !in1F2)

	// F3 = OR
	in1F3 := applyFault("F3", "in1", x[5])
	in2F3 := applyFault("F3", "in2", x[6])
	in3F3 := applyFault("F3", "in3", x[7])

	f3 = applyFault(
		"F3", "out",
		in1F3 || in2F3 || in3F3,
	)

	// F4 = AND
	in1F4 := applyFault("F4", "in1", f3)
	in2F4 := applyFault("F4", "in2", x[4])

	f4 = applyFault(
		"F4", "out",
		in1F4 && in2F4,
	)

	// F5 = NAND
	in1F5 := applyFault("F5", "in1", f2)
	in2F5 := applyFault("F5", "in2", f4)

	f5 = applyFault(
		"F5", "out",
		!(in1F5 && in2F5),
	)

	// F6 = AND
	in1F6 := applyFault("F6", "in1", f1)
	in2F6 := applyFault("F6", "in2", f5)

	f6 = applyFault(
		"F6", "out",
		in1F6 && in2F6,
	)

	return
}

// Проверка, проявляется ли неисправность
// на внешнем выходе F6 при заданном тестовом наборе.

func detectFault(x [8]bool, f Fault) bool {
	_, _, _, _, _, good := evaluate(x)
	_, _, _, _, _, bad := evaluateFaulty(x, f)

	return good != bad
}

// ============================================================
// 5. Преобразование состояния LFSR в тестовый набор
// ============================================================

// В соответствии с ЛР №3:
// x1 = bit 0, x2 = bit 1, ..., x7 = bit 6.

func stateToInputs(state uint8) [8]bool {
	var x [8]bool

	for j := 1; j <= 7; j++ {
		x[j] = ((state >> uint(j-1)) & 1) == 1
	}

	return x
}

// ============================================================
// 6. Поиск минимального теста
// ============================================================

type TestResult struct {
	Tacts       int
	Seed        uint8
	TestVectors []uint8
}

func findMinTest(
	lfsr *LFSR,
	maxTacts int,
) *TestResult {
	faults := allFaults()
	totalFaults := len(faults)

	var best *TestResult

	// Перебираются все ненулевые 8-разрядные seed.
	for seed := 1; seed <= 255; seed++ {
		lfsr.Reset(uint8(seed))

		covered := make([]bool, totalFaults)
		coveredCount := 0
		var vectors []uint8

		for tact := 1; tact <= maxTacts; tact++ {
			state := lfsr.State()
			vectors = append(vectors, state)

			x := stateToInputs(state)

			for k, fault := range faults {
				if !covered[k] && detectFault(x, fault) {
					covered[k] = true
					coveredCount++
				}
			}

			if coveredCount == totalFaults {
				if best == nil || tact < best.Tacts {
					vecCopy := append([]uint8(nil), vectors...)

					best = &TestResult{
						Tacts:       tact,
						Seed:        uint8(seed),
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

// ============================================================
// 7. Формирование последовательностей выходных реакций
// ============================================================

func outputSequence(
	vectors []uint8,
	fault *Fault,
) []bool {
	result := make([]bool, len(vectors))

	for i, state := range vectors {
		x := stateToInputs(state)

		if fault == nil {
			_, _, _, _, _, result[i] = evaluate(x)
		} else {
			_, _, _, _, _, result[i] = evaluateFaulty(x, *fault)
		}
	}

	return result
}

// ============================================================
// 8. Модель сигнатурного анализатора
// ============================================================

// taps задаёт младшие члены полинома.
// Например:
// degree = 3, taps = 0b011
// соответствует x^3 + x + 1.
//
// Регистр реализует деление последовательности на
// порождающий полином над GF(2).

func signature(
	sequence []bool,
	degree int,
	taps int,
) int {
	mask := (1 << degree) - 1
	topBit := 1 << (degree - 1)

	reg := 0

	for _, input := range sequence {
		inputBit := 0
		if input {
			inputBit = 1
		}

		feedback := ((reg & topBit) >> (degree - 1)) ^ inputBit

		reg = (reg << 1) & mask

		if feedback != 0 {
			reg ^= taps
		}
	}

	return reg
}

// ============================================================
// 9. Оценка обнаружения неисправностей сигнатурным анализатором
// ============================================================

type Polynomial struct {
	Degree   int
	Taps     int
	XORCount int
	Detected int
	Total    int
	Coverage float64
}

func evaluatePolynomial(
	vectors []uint8,
	faults []Fault,
	degree int,
	taps int,
) Polynomial {
	goodSequence := outputSequence(vectors, nil)
	goodSignature := signature(goodSequence, degree, taps)

	detected := 0

	for _, fault := range faults {
		badSequence := outputSequence(vectors, &fault)
		badSignature := signature(badSequence, degree, taps)

		if badSignature != goodSignature {
			detected++
		}
	}

	total := len(faults)
	coverage := 100.0 * float64(detected) / float64(total)

	// Число двухвходовых XOR для суммирования
	// членов обратной связи при принятой модели подсчёта.
	xorCount := bits.OnesCount(uint(taps)) - 1

	return Polynomial{
		Degree:   degree,
		Taps:     taps,
		XORCount: xorCount,
		Detected: detected,
		Total:    total,
		Coverage: coverage,
	}
}

// ============================================================
// 10. Перебор порождающих полиномов
// ============================================================

func findPolynomials(
	vectors []uint8,
	faults []Fault,
	degree int,
) []Polynomial {
	var results []Polynomial

	// Старший член x^degree подразумевается.
	// Постоянный член должен быть равен 1.
	//
	// Перебираются все комбинации младших членов,
	// у которых установлен бит x^0.

	for taps := 1; taps < (1 << degree); taps += 2 {
		p := evaluatePolynomial(
			vectors,
			faults,
			degree,
			taps,
		)

		if p.Detected == p.Total {
			results = append(results, p)
		}
	}

	// Сначала минимальное число XOR,
	// затем лексикографически по маске taps.
	sort.Slice(results, func(i, j int) bool {
		if results[i].XORCount != results[j].XORCount {
			return results[i].XORCount < results[j].XORCount
		}

		return results[i].Taps < results[j].Taps
	})

	return results
}

// ============================================================
// 11. Вывод полинома в символьном виде
// ============================================================

func polynomialString(degree, taps int) string {
	terms := []string{
		fmt.Sprintf("x^%d", degree),
	}

	for power := degree - 1; power >= 0; power-- {
		if taps&(1<<power) == 0 {
			continue
		}

		switch power {
		case 0:
			terms = append(terms, "1")
		case 1:
			terms = append(terms, "x")
		default:
			terms = append(
				terms,
				fmt.Sprintf("x^%d", power),
			)
		}
	}

	return strings.Join(terms, " + ")
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// ============================================================
// 12. Главная функция
// ============================================================

func main() {
	fmt.Println("==============================================")
	fmt.Printf("Лабораторная работа №5, вариант %d\n", VariantNumber)
	fmt.Println("Компактное тестирование цифровой схемы")
	fmt.Println("==============================================")

	fmt.Printf("Полином LFSR: %s\n", PolyString)
	fmt.Printf("Степень LFSR: %d\n", LFSRDegree)
	fmt.Printf("Отводы LFSR: %v\n\n", []int{5, 3, 1, 0})

	lfsr := NewLFSR(
		[]int{5, 3, 1, 0},
		LFSRDegree,
		1,
	)

	faults := allFaults()

	fmt.Printf("Количество одиночных неисправностей: %d\n", len(faults))

	// --------------------------------------------------------
	// Часть 1. Поиск минимального теста
	// --------------------------------------------------------

	fmt.Println("\n1. Поиск минимального теста")

	result := findMinTest(lfsr, MaxTacts)

	if result == nil {
		fmt.Println("Не удалось получить 100% покрытие тестом.")
		return
	}

	fmt.Printf("Количество тактов: %d\n", result.Tacts)
	fmt.Printf("Начальное состояние: %08b (0x%02X)\n",
		result.Seed, result.Seed)

	// Повторная проверка покрытия найденного теста.
	covered := make([]bool, len(faults))
	coveredCount := 0

	for _, state := range result.TestVectors {
		x := stateToInputs(state)

		for i, fault := range faults {
			if !covered[i] && detectFault(x, fault) {
				covered[i] = true
				coveredCount++
			}
		}
	}

	fmt.Printf("Обнаружено неисправностей: %d/%d\n",
		coveredCount, len(faults))
	fmt.Printf("Покрытие: %.2f%%\n",
		100.0*float64(coveredCount)/float64(len(faults)))

	fmt.Println("\nТестовая последовательность:")
	fmt.Printf("%-6s %-10s %-10s %-5s\n",
		"Такт", "LFSR", "x1..x7", "F6")

	for i, state := range result.TestVectors {
		x := stateToInputs(state)
		_, _, _, _, _, f6 := evaluate(x)

		var inputString strings.Builder
		for j := 1; j <= 7; j++ {
			if x[j] {
				inputString.WriteByte('1')
			} else {
				inputString.WriteByte('0')
			}
		}

		fmt.Printf("%-6d %08b   %-10s %-5d\n",
			i,
			state,
			inputString.String(),
			boolToInt(f6),
		)
	}

	// --------------------------------------------------------
	// Часть 2. Подбор полиномов сигнатурного анализатора
	// --------------------------------------------------------

	fmt.Println("\n2. Подбор полиномов сигнатурного анализатора")
	fmt.Println("Условие: обнаружение всех одиночных неисправностей.")
	fmt.Println()

	fmt.Printf(
		"%-5s %-30s %-12s %-15s\n",
		"m",
		"Полином",
		"Число XOR",
		"Покрытие",
	)

	for _, degree := range []int{7, 8, 9} {
		results := findPolynomials(
			result.TestVectors,
			faults,
			degree,
		)

		if len(results) == 0 {
			fmt.Printf(
				"%-5d %-30s %-12s %-15s\n",
				degree,
				"Не найден",
				"-",
				"Менее 100%",
			)
			continue
		}

		// Первый элемент имеет минимальную аппаратную сложность.
		best := results[0]

		fmt.Printf(
			"%-5d %-30s %-12d %d/%d (%.2f%%)\n",
			best.Degree,
			polynomialString(best.Degree, best.Taps),
			best.XORCount,
			best.Detected,
			best.Total,
			best.Coverage,
		)

		fmt.Printf(
			"      Найдено полиномов с 100%% покрытием: %d\n",
			len(results),
		)
	}

	fmt.Println("\n==============================================")
	fmt.Println("Моделирование завершено.")
	fmt.Println("==============================================")
}
