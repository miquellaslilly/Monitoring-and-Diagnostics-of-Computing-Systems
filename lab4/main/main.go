package main

import (
	"bufio"
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	VariantNumber = 9
	LFSRDegree    = 8
	MaxLFSRState  = (1 << LFSRDegree) - 1
)

var generatorTaps = []int{5, 3, 1, 0}

// ------------------------------------------------------------
// 1. Генератор тестовых воздействий на основе LFSR
// ------------------------------------------------------------

type LFSR struct {
	state uint8
}

func NewLFSR(seed uint8) *LFSR {
	if seed == 0 {
		seed = 1
	}
	return &LFSR{state: seed}
}

func (l *LFSR) State() uint8 { return l.state }

func (l *LFSR) Next() uint8 {
	var feedback uint8
	for _, tap := range generatorTaps {
		feedback ^= (l.state >> tap) & 1
	}
	l.state = (l.state >> 1) | (feedback << (LFSRDegree - 1))
	return l.state
}

type TestVector struct {
	Tact   int
	State  uint8
	Inputs [7]bool
}

func stateToInputs(state uint8) [7]bool {
	var inputs [7]bool
	for i := 0; i < 7; i++ {
		inputs[i] = ((state >> i) & 1) != 0
	}
	return inputs
}

func makeVector(tact int, state uint8) TestVector {
	return TestVector{Tact: tact, State: state, Inputs: stateToInputs(state)}
}

// ------------------------------------------------------------
// 2. Модель одиночных константных неисправностей
// ------------------------------------------------------------

type Fault struct {
	Gate  string
	Pin   string
	Value bool
}

func (f Fault) String() string {
	value := 0
	if f.Value {
		value = 1
	}
	return fmt.Sprintf("%s.%s stuck-at-%d", f.Gate, f.Pin, value)
}

func allFaults() []Fault {
	var faults []Fault
	addGateFaults(&faults, "F1", []string{"in1", "in2", "out"})
	addGateFaults(&faults, "F2", []string{"in1", "out"})
	addGateFaults(&faults, "F3", []string{"in1", "in2", "in3", "out"})
	addGateFaults(&faults, "F4", []string{"in1", "in2", "out"})
	addGateFaults(&faults, "F5", []string{"in1", "in2", "out"})
	addGateFaults(&faults, "F6", []string{"in1", "in2", "out"})
	return faults
}

func addGateFaults(faults *[]Fault, gate string, pins []string) {
	for _, pin := range pins {
		*faults = append(*faults,
			Fault{Gate: gate, Pin: pin, Value: false},
			Fault{Gate: gate, Pin: pin, Value: true},
		)
	}
}

func faultyInput(fault Fault, gate, pin string, original bool) bool {
	if fault.Gate == gate && fault.Pin == pin {
		return fault.Value
	}
	return original
}

func faultyOutput(fault Fault, gate string, original bool) bool {
	if fault.Gate == gate && fault.Pin == "out" {
		return fault.Value
	}
	return original
}

// ------------------------------------------------------------
// 3. Исправная и неисправная схемы
// ------------------------------------------------------------

func evaluate(inputs [7]bool) bool {
	f1 := inputs[0] && inputs[1]
	f2 := !inputs[2]
	f3 := inputs[4] || inputs[5] || inputs[6]
	f4 := f3 && inputs[3]
	f5 := !(f2 && f4)
	f6 := f1 && f5
	return f6
}

func evaluateFaulty(inputs [7]bool, fault Fault) bool {
	f1in1 := faultyInput(fault, "F1", "in1", inputs[0])
	f1in2 := faultyInput(fault, "F1", "in2", inputs[1])
	f1 := faultyOutput(fault, "F1", f1in1 && f1in2)

	f2in1 := faultyInput(fault, "F2", "in1", inputs[2])
	f2 := faultyOutput(fault, "F2", !f2in1)

	f3in1 := faultyInput(fault, "F3", "in1", inputs[4])
	f3in2 := faultyInput(fault, "F3", "in2", inputs[5])
	f3in3 := faultyInput(fault, "F3", "in3", inputs[6])
	f3 := faultyOutput(fault, "F3", f3in1 || f3in2 || f3in3)

	f4in1 := faultyInput(fault, "F4", "in1", f3)
	f4in2 := faultyInput(fault, "F4", "in2", inputs[3])
	f4 := faultyOutput(fault, "F4", f4in1 && f4in2)

	f5in1 := faultyInput(fault, "F5", "in1", f2)
	f5in2 := faultyInput(fault, "F5", "in2", f4)
	f5 := faultyOutput(fault, "F5", !(f5in1 && f5in2))

	f6in1 := faultyInput(fault, "F6", "in1", f1)
	f6in2 := faultyInput(fault, "F6", "in2", f5)
	f6 := faultyOutput(fault, "F6", f6in1 && f6in2)

	return f6
}

// ------------------------------------------------------------
// 4. Поиск тестовой последовательности
// ------------------------------------------------------------

func detectFault(v TestVector, fault Fault) bool {
	good := evaluate(v.Inputs)
	bad := evaluateFaulty(v.Inputs, fault)
	return good != bad
}

func findMinTest(faults []Fault) ([]TestVector, uint8, bool) {
	var best []TestVector
	var bestSeed uint8

	for seed := 1; seed <= MaxLFSRState; seed++ {
		lfsr := NewLFSR(uint8(seed))
		detected := make([]bool, len(faults))
		detectedCount := 0
		var vectors []TestVector

		for tact := 1; tact <= MaxLFSRState; tact++ {
			state := lfsr.State()
			vector := makeVector(tact, state)
			vectors = append(vectors, vector)

			for i, fault := range faults {
				if !detected[i] && detectFault(vector, fault) {
					detected[i] = true
					detectedCount++
				}
			}

			if detectedCount == len(faults) {
				if best == nil || len(vectors) < len(best) {
					best = append([]TestVector(nil), vectors...)
					bestSeed = uint8(seed)
				}
				break
			}
			lfsr.Next()
		}
	}
	return best, bestSeed, best != nil
}

// ------------------------------------------------------------
// 5. Выходные последовательности
// ------------------------------------------------------------

func outputSequence(vectors []TestVector, fault *Fault) []bool {
	result := make([]bool, 0, len(vectors))
	for _, vector := range vectors {
		var output bool
		if fault == nil {
			output = evaluate(vector.Inputs)
		} else {
			output = evaluateFaulty(vector.Inputs, *fault)
		}
		result = append(result, output)
	}
	return result
}

// ------------------------------------------------------------
// 6. Сигнатурный анализатор
// ------------------------------------------------------------

func signature(sequence []bool, degree int, lowerMask uint16) uint16 {
	var register uint16
	mask := uint16((1 << degree) - 1)
	topBit := uint16(1 << (degree - 1))

	for _, input := range sequence {
		inputBit := uint16(0)
		if input {
			inputBit = 1
		}
		feedback := ((register & topBit) != 0) != (inputBit != 0)
		register = (register << 1) & mask
		if feedback {
			register ^= lowerMask
		}
	}
	return register
}

// ------------------------------------------------------------
// 7. Перебор полиномов
// ------------------------------------------------------------

type PolynomialResult struct {
	Degree        int
	LowerMask     uint16
	Polynomial    string
	XORGates      int
	Detected      int
	TotalFaults   int
	Coverage      float64
	GoodSignature uint16
}

func polynomialString(degree int, lowerMask uint16) string {
	terms := []string{fmt.Sprintf("x^%d", degree)}
	for power := degree - 1; power >= 0; power-- {
		if lowerMask&(1<<power) == 0 {
			continue
		}
		if power == 0 {
			terms = append(terms, "1")
		} else if power == 1 {
			terms = append(terms, "x")
		} else {
			terms = append(terms, fmt.Sprintf("x^%d", power))
		}
	}
	return strings.Join(terms, " ⊕ ")
}

func hardwareCost(lowerMask uint16) int {
	count := bits.OnesCount16(lowerMask) - 1
	if count < 0 {
		return 0
	}
	return count
}

func evaluatePolynomial(
	degree int,
	lowerMask uint16,
	goodSequence []bool,
	faultSequences [][]bool,
) PolynomialResult {
	goodSignature := signature(goodSequence, degree, lowerMask)
	detected := 0

	for _, faultySequence := range faultSequences {
		if signature(faultySequence, degree, lowerMask) != goodSignature {
			detected++
		}
	}

	total := len(faultSequences)
	coverage := 0.0
	if total > 0 {
		coverage = 100.0 * float64(detected) / float64(total)
	}

	return PolynomialResult{
		Degree:        degree,
		LowerMask:     lowerMask,
		Polynomial:    polynomialString(degree, lowerMask),
		XORGates:      hardwareCost(lowerMask),
		Detected:      detected,
		TotalFaults:   total,
		Coverage:      coverage,
		GoodSignature: goodSignature,
	}
}

func findPolynomials(degree int, goodSequence []bool, faultSequences [][]bool) []PolynomialResult {
	var results []PolynomialResult
	limit := 1 << degree
	for lowerMask := 1; lowerMask < limit; lowerMask += 2 {
		results = append(results, evaluatePolynomial(degree, uint16(lowerMask), goodSequence, faultSequences))
	}
	sort.Slice(results, func(i, j int) bool {
		iFull := results[i].Detected == results[i].TotalFaults
		jFull := results[j].Detected == results[j].TotalFaults
		if iFull != jFull {
			return iFull
		}
		if results[i].XORGates != results[j].XORGates {
			return results[i].XORGates < results[j].XORGates
		}
		if results[i].Coverage != results[j].Coverage {
			return results[i].Coverage > results[j].Coverage
		}
		return results[i].LowerMask < results[j].LowerMask
	})
	return results
}

// ------------------------------------------------------------
// 8. Вывод таблицы результатов
// ------------------------------------------------------------

func printPolynomialResults(degree int, results []PolynomialResult) {
	fmt.Printf("\n=== Сигнатурный анализатор: m = %d ===\n", degree)
	fmt.Printf("%-42s %-12s %-14s %-12s\n",
		"Порождающий полином", "XOR-сумматоры", "Обнаружено", "Покрытие")

	bestFound := false
	for _, result := range results {
		fmt.Printf("%-42s %-12d %d/%d           %6.2f%%\n",
			result.Polynomial, result.XORGates,
			result.Detected, result.TotalFaults, result.Coverage)

		if result.Detected == result.TotalFaults && !bestFound {
			fmt.Printf("Лучший полином для m=%d: %s\n", degree, result.Polynomial)
			fmt.Printf("Аппаратная сложность: %d XOR-сумматоров\n", result.XORGates)
			fmt.Printf("Эталонная сигнатура: %0*b\n", degree, result.GoodSignature)
			bestFound = true
		}
	}
	if !bestFound {
		fmt.Println("Полином со 100% обнаружением в этой модели не найден.")
	}
}

// ------------------------------------------------------------
// 9. Выбор неисправности и подробный вывод по ней
// ------------------------------------------------------------

func printFaultList(faults []Fault) {
	fmt.Println("\n=== Список моделируемых неисправностей ===")
	for i, f := range faults {
		fmt.Printf("%2d) %s\n", i+1, f.String())
	}
}

func printBits(label string, seq []bool) {
	fmt.Printf("%s (%d бит):\n", label, len(seq))
	// печатаем группами по 32 бита для читаемости
	for i := 0; i < len(seq); i += 32 {
		end := i + 32
		if end > len(seq) {
			end = len(seq)
		}
		fmt.Printf("  такты %3d..%3d: ", i+1, end)
		for _, b := range seq[i:end] {
			if b {
				fmt.Print("1")
			} else {
				fmt.Print("0")
			}
		}
		fmt.Println()
	}
}

func showFaultDetails(
	fault Fault,
	vectors []TestVector,
	goodSequence []bool,
	faultSequence []bool,
	bestResults map[int]PolynomialResult,
) {
	fmt.Println("\n============================================================")
	fmt.Printf("Выбранная неисправность: %s\n", fault.String())
	fmt.Println("============================================================")

	// 1. Сравнение выходных последовательностей
	diffs := []int{}
	for i := range goodSequence {
		if goodSequence[i] != faultSequence[i] {
			diffs = append(diffs, i+1) // такты 1-based
		}
	}

	if len(diffs) == 0 {
		fmt.Println("Выходные последовательности СОВПАДАЮТ — неисправность НЕ обнаружена.")
		return
	}

	fmt.Printf("Обнаружена прямым сравнением: ДА (%d расхождений из %d тактов)\n",
		len(diffs), len(goodSequence))
	fmt.Printf("Такты с расхождением: %v\n", diffs)

	// 2. Побитовый вывод последовательностей (сокращённо — первые 64 бита и расхождения)
	fmt.Println("\n--- Выходная последовательность F6 ---")
	printBits("Эталон (исправная схема)", goodSequence)
	printBits("Неисправная схема       ", faultSequence)

	// 3. Сигнатуры для m=7,8,9 по лучшим полиномам
	fmt.Println("\n--- Сигнатуры для лучших полиномов ---")
	for _, m := range []int{7, 8, 9} {
		res, ok := bestResults[m]
		if !ok {
			continue
		}
		goodSig := signature(goodSequence, m, res.LowerMask)
		badSig := signature(faultSequence, m, res.LowerMask)
		status := "ОБНАРУЖЕНО"
		if goodSig == badSig {
			status = "ЗАМАСКИРОВАНО"
		}
		fmt.Printf("m=%d, полином %-42s | эталон=%0*b  неиспр=%0*b  → %s\n",
			m, res.Polynomial, m, goodSig, m, badSig, status)
	}
}

func readFaultChoice(reader *bufio.Reader, max int) (int, bool) {
	fmt.Printf("\nВведите номер неисправности (1..%d) или 0 для выхода: ", max)
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, false
	}
	line = strings.TrimSpace(line)
	n, err := strconv.Atoi(line)
	if err != nil || n < 0 || n > max {
		fmt.Println("Некорректный ввод.")
		return 0, false
	}
	return n, true
}

// ------------------------------------------------------------
// 10. Главная функция
// ------------------------------------------------------------

func main() {
	fmt.Println("Лабораторная работа №4. Вариант", VariantNumber)
	fmt.Println("Компактное тестирование: ЛФСР -> схема -> сигнатурный анализатор")

	faults := allFaults()
	fmt.Printf("\nВсего моделируемых одиночных константных неисправностей: %d\n", len(faults))

	vectors, seed, ok := findMinTest(faults)
	if !ok {
		fmt.Println("Не удалось найти тестовую последовательность.")
		return
	}

	fmt.Printf("Найдено тестовых тактов: %d\n", len(vectors))
	fmt.Printf("Начальное состояние ЛФСР: %08b (0x%02X)\n", seed, seed)

	fmt.Println("\nТестовые наборы:")
	fmt.Printf("%-6s %-10s %-7s %-7s %-7s %-7s %-7s %-7s %-7s\n",
		"Такт", "Состояние", "x1", "x2", "x3", "x4", "x5", "x6", "x7")
	for _, v := range vectors {
		fmt.Printf("%-6d %-10s", v.Tact, fmt.Sprintf("%08b", v.State))
		for _, input := range v.Inputs {
			value := 0
			if input {
				value = 1
			}
			fmt.Printf(" %-7d", value)
		}
		fmt.Println()
	}

	goodSequence := outputSequence(vectors, nil)
	faultSequences := make([][]bool, 0, len(faults))
	for _, fault := range faults {
		faultSequences = append(faultSequences, outputSequence(vectors, &fault))
	}

	outputDetected := 0
	for _, fault := range faults {
		for _, v := range vectors {
			if detectFault(v, fault) {
				outputDetected++
				break
			}
		}
	}
	fmt.Printf("\nОбнаружено сравнением выходных последовательностей: %d/%d (%.2f%%)\n",
		outputDetected, len(faults),
		100.0*float64(outputDetected)/float64(len(faults)))

	// Для каждой степени находим лучший полином и запоминаем
	bestResults := map[int]PolynomialResult{}
	for _, degree := range []int{7, 8, 9} {
		results := findPolynomials(degree, goodSequence, faultSequences)
		printPolynomialResults(degree, results)
		// первый в отсортированном списке с 100% покрытием
		for _, r := range results {
			if r.Detected == r.TotalFaults {
				bestResults[degree] = r
				break
			}
		}
	}

	// --------------------------------------------------------
	// Интерактивный выбор неисправности
	// --------------------------------------------------------
	printFaultList(faults)

	reader := bufio.NewReader(os.Stdin)
	for {
		n, ok := readFaultChoice(reader, len(faults))
		if !ok {
			continue
		}
		if n == 0 {
			fmt.Println("Выход.")
			return
		}

		idx := n - 1
		showFaultDetails(
			faults[idx],
			vectors,
			goodSequence,
			faultSequences[idx],
			bestResults,
		)
	}
}
