package main

import (
	"fmt"
)

const (
	Variant    = 9
	Length     = 44   // l из варианта 9
	Degree     = 8    // степень полинома
	LowerMask  = 0x2B // x^8 + x^5 + x^3 + x + 1
	Polynomial = "x^8 ⊕ x^5 ⊕ x^3 ⊕ x ⊕ 1"
)

// ------------------------------------------------------------
// Ядро: LFSR-деление на полином (одноканальный ОСА)
// ------------------------------------------------------------

func signature(bits []bool, lowerMask uint16, degree int) uint16 {
	var reg uint16
	mask := uint16((1 << degree) - 1)
	top := uint16(1 << (degree - 1))

	for _, b := range bits {
		in := uint16(0)
		if b {
			in = 1
		}
		fb := ((reg & top) != 0) != (in != 0)
		reg = (reg << 1) & mask
		if fb {
			reg ^= lowerMask
		}
	}
	return reg
}

// Кодирование позиций ошибок (1-based) в битовый вектор длины l
func errorVector(positions []int, length int) []bool {
	v := make([]bool, length)
	for _, p := range positions {
		v[p-1] = true
	}
	return v
}

// ------------------------------------------------------------
// Перебор ошибок кратности k: возвращает счётчики и примеры
// ------------------------------------------------------------

type ErrorSearch struct {
	Multiplicity int
	Total        int
	Undetected   int
	Examples     [][]int // примеры необнаруженных (позиции)
}

// combinations рекурсивно генерирует сочетания
func searchErrors(k int, length int, lowerMask uint16) ErrorSearch {
	res := ErrorSearch{Multiplicity: k}
	pos := make([]int, k)

	var rec func(start, depth int)
	rec = func(start, depth int) {
		if depth == k {
			res.Total++
			bitsVec := errorVector(pos, length)
			if signature(bitsVec, lowerMask, Degree) == 0 {
				res.Undetected++
				if len(res.Examples) < 10 {
					ex := make([]int, k)
					copy(ex, pos)
					res.Examples = append(res.Examples, ex)
				}
			}
			return
		}
		for p := start; p <= length-(k-depth)+1; p++ {
			pos[depth] = p
			rec(p+1, depth+1)
		}
	}
	rec(1, 0)
	return res
}

func printSearch(r ErrorSearch) {
	cover := 0.0
	if r.Total > 0 {
		cover = 100.0 * float64(r.Total-r.Undetected) / float64(r.Total)
	}
	fmt.Printf("Кратность %d: всего %d, не обнаружено %d, покрытие %.2f%%\n",
		r.Multiplicity, r.Total, r.Undetected, cover)
	if r.Undetected > 0 {
		fmt.Print("  Примеры необнаруженных (позиции): ")
		for i, ex := range r.Examples {
			if i > 0 {
				fmt.Print("; ")
			}
			fmt.Print(ex)
		}
		fmt.Println()
	}
}

// ------------------------------------------------------------
// Двухканальный анализатор
//
// Идея: последовательность делится на чётные и нечётные биты,
// каждый канал — свой ОСА. Используем два полинома: наш основной
// и ещё один (например, x^8 + x^4 + x^3 + x^2 + 1 — примитивный).
// Сигнатура двухканального = пара (s1, s2).
// Ошибка не обнаружена, только если ОБА остатка нулевые.
// ------------------------------------------------------------

const (
	DualMask1 = 0x2B // x^8+x^5+x^3+x+1  (вариант 9)
	DualMask2 = 0x1D // x^8+x^4+x^3+x^2+1 (пример примитивного)
)

// Разделение последовательности на чётные/нечётные позиции (0-based)
func splitEvenOdd(bits []bool) ([]bool, []bool) {
	even := make([]bool, 0, (len(bits)+1)/2)
	odd := make([]bool, 0, len(bits)/2)
	for i, b := range bits {
		if i%2 == 0 {
			even = append(even, b)
		} else {
			odd = append(odd, b)
		}
	}
	return even, odd
}

func dualSignature(bits []bool, mask1, mask2 uint16) (uint16, uint16) {
	even, odd := splitEvenOdd(bits)
	return signature(even, mask1, Degree), signature(odd, mask2, Degree)
}

func searchErrorsDual(k int, length int, mask1, mask2 uint16) ErrorSearch {
	res := ErrorSearch{Multiplicity: k}
	pos := make([]int, k)

	var rec func(start, depth int)
	rec = func(start, depth int) {
		if depth == k {
			res.Total++
			bitsVec := errorVector(pos, length)
			s1, s2 := dualSignature(bitsVec, mask1, mask2)
			if s1 == 0 && s2 == 0 {
				res.Undetected++
				if len(res.Examples) < 10 {
					ex := make([]int, k)
					copy(ex, pos)
					res.Examples = append(res.Examples, ex)
				}
			}
			return
		}
		for p := start; p <= length-(k-depth)+1; p++ {
			pos[depth] = p
			rec(p+1, depth+1)
		}
	}
	rec(1, 0)
	return res
}

// ------------------------------------------------------------
// main
// ------------------------------------------------------------

func main() {
	fmt.Printf("Лабораторная работа №6, вариант %d\n", Variant)
	fmt.Printf("Полином: %s\n", Polynomial)
	fmt.Printf("Длина сжимаемой последовательности l = %d\n", Length)
	fmt.Printf("Маска (lowerMask) = 0x%02X = %08b\n\n", LowerMask, LowerMask)

	// Пункт 1. Одноканальный ОСА
	fmt.Println("=== Пункт 1. Одноканальный сигнатурный анализатор ===")
	var results [5]ErrorSearch
	for k := 1; k <= 4; k++ {
		results[k] = searchErrors(k, Length, LowerMask)
		printSearch(results[k])
	}

	// Демонстрация сигнатур на произвольной последовательности
	fmt.Println("\nДемонстрация: сигнатура для случайной последовательности")
	sample := make([]bool, Length)
	for i := 0; i < Length; i++ {
		sample[i] = ((i*7 + 3) % 5) < 2
	}
	fmt.Printf("  последовательность: ")
	for _, b := range sample {
		if b {
			fmt.Print("1")
		} else {
			fmt.Print("0")
		}
	}
	fmt.Printf("\n  сигнатура = 0x%02X = %08b\n", signature(sample, LowerMask, Degree), signature(sample, LowerMask, Degree))

	// Пункт 2. Двухканальный анализатор
	fmt.Println("\n=== Пункт 2. Двухканальный сигнатурный анализатор ===")
	fmt.Printf("Канал 1: полином 1, маска 0x%02X\n", DualMask1)
	fmt.Printf("Канал 2: полином 2, маска 0x%02X\n", DualMask2)
	fmt.Println("Разделение: чётные биты → канал 1, нечётные → канал 2")
	fmt.Println()

	var resultsDual [5]ErrorSearch
	for k := 1; k <= 4; k++ {
		resultsDual[k] = searchErrorsDual(k, Length, DualMask1, DualMask2)
		printSearch(resultsDual[k])
	}

	// Сравнение
	fmt.Println("\n=== Сравнение одноканального и двухканального ===")
	fmt.Printf("%-12s %-25s %-25s\n", "Кратность", "1 канал (обнаружено/всего)", "2 канала (обнаружено/всего)")
	for k := 1; k <= 4; k++ {
		o := results[k].Total - results[k].Undetected
		d := resultsDual[k].Total - resultsDual[k].Undetected
		fmt.Printf("%-12d %-25s %-25s\n",
			k,
			fmt.Sprintf("%d/%d", o, results[k].Total),
			fmt.Sprintf("%d/%d", d, resultsDual[k].Total))
	}
}
