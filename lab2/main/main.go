package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Value string

const (
	Zero Value = "0"
	One  Value = "1"
	X    Value = "X"
	D    Value = "d"
	DB   Value = "d'"
)

type Cube map[string]Value

type Gate struct {
	Name, Type string
	Inputs     []string
	Output     string
}

type Fault struct {
	Node    string
	StuckAt int
}

type TraceStep struct {
	Before   Cube
	Desc     string
	UsedCube Cube
	Cube     Cube
}

var poles = []string{
	"x1", "x2", "x3", "x4", "x5", "x6", "x7",
	"F1", "F2", "F3", "F4", "F5", "F6",
}

var poleNo = map[string]int{
	"x1": 1,
	"x2": 2,
	"x3": 3,
	"x4": 4,
	"x5": 5,
	"x6": 6,
	"x7": 7,
	"F1": 8,
	"F2": 9,
	"F3": 10,
	"F4": 11,
	"F5": 12,
	"F6": 13,
}

var gates = []Gate{
	{"F1", "AND", []string{"x1", "x2"}, "F1"},
	{"F2", "NOT", []string{"x3"}, "F2"},
	{"F3", "OR", []string{"x5", "x6"}, "F3"},
	{"F4", "AND", []string{"x4", "F3", "x7"}, "F4"},
	{"F5", "NAND", []string{"F2", "F4"}, "F5"},
	{"F6", "AND", []string{"F1", "F5"}, "F6"},
}

func gateOf(out string) *Gate {
	for i := range gates {
		if gates[i].Output == out {
			return &gates[i]
		}
	}
	return nil
}

func copyCube(c Cube) Cube {
	r := Cube{}

	for k, v := range c {
		r[k] = v
	}

	return r
}

func merge(a, b Cube) (Cube, bool) {
	r := copyCube(a)

	for k, v := range b {
		old, ok := r[k]

		if !ok || old == X {
			r[k] = v
		} else if v == X || old == v {
			// Совместимо.
		} else {
			return nil, false
		}
	}

	return r, true
}

func fmtSeq(c Cube) string {
	var b strings.Builder

	for _, p := range poles {
		v := c[p]

		if v == "" {
			v = X
		}

		b.WriteString(string(v))
	}

	return b.String()
}

func printCube(c Cube) {
	for _, p := range poles {
		v := c[p]

		if v == "" {
			v = X
		}

		fmt.Printf("%2s ", v)
	}

	fmt.Println()
}

func printCubeInline(c Cube) {
	for _, p := range poles {
		v := c[p]

		if v == "" {
			v = X
		}

		fmt.Printf("%s", v)
	}
}

func primitive(f Fault) Cube {
	if f.StuckAt == 0 {
		return Cube{f.Node: D}
	}

	return Cube{f.Node: DB}
}

// ============================================================
// D-КУБЫ
// ============================================================

func dCubes(g Gate) []Cube {
	out := []Cube{}

	switch g.Type {

	case "AND":
		for _, dv := range []Value{D, DB} {
			for i := range g.Inputs {

				c := Cube{}

				for j, n := range g.Inputs {
					c[n] = One

					if i == j {
						c[n] = dv
					}
				}

				c[g.Output] = dv
				out = append(out, c)
			}
		}

	case "OR":
		for _, dv := range []Value{D, DB} {
			for i := range g.Inputs {

				c := Cube{}

				for j, n := range g.Inputs {
					c[n] = Zero

					if i == j {
						c[n] = dv
					}
				}

				c[g.Output] = dv
				out = append(out, c)
			}
		}

	case "NAND":
		for _, dv := range []Value{D, DB} {
			for i := range g.Inputs {

				c := Cube{}

				for j, n := range g.Inputs {
					c[n] = One

					if i == j {
						c[n] = dv
					}
				}

				if dv == D {
					c[g.Output] = DB
				} else {
					c[g.Output] = D
				}

				out = append(out, c)
			}
		}

	case "NOT":
		out = append(
			out,
			Cube{
				g.Inputs[0]: D,
				g.Output:    DB,
			},
			Cube{
				g.Inputs[0]: DB,
				g.Output:    D,
			},
		)
	}

	return out
}

// ============================================================
// СИНГУЛЯРНЫЕ КУБЫ
// ============================================================

func singularCubes(g Gate, desired Value) []Cube {
	out := []Cube{}

	switch g.Type {

	case "AND":

		if desired == One {
			c := Cube{
				g.Output: One,
			}

			for _, in := range g.Inputs {
				c[in] = One
			}

			out = append(out, c)
		}

		if desired == Zero {
			for _, in := range g.Inputs {

				c := Cube{
					g.Output: Zero,
				}

				for _, n := range g.Inputs {
					c[n] = X
				}

				c[in] = Zero

				out = append(out, c)
			}
		}

	case "OR":

		if desired == Zero {
			c := Cube{
				g.Output: Zero,
			}

			for _, in := range g.Inputs {
				c[in] = Zero
			}

			out = append(out, c)
		}

		if desired == One {
			for _, in := range g.Inputs {

				c := Cube{
					g.Output: One,
				}

				for _, n := range g.Inputs {
					c[n] = X
				}

				c[in] = One

				out = append(out, c)
			}
		}

	case "NAND":

		if desired == Zero {
			c := Cube{
				g.Output: Zero,
			}

			for _, in := range g.Inputs {
				c[in] = One
			}

			out = append(out, c)
		}

		if desired == One {
			for _, in := range g.Inputs {

				c := Cube{
					g.Output: One,
				}

				for _, n := range g.Inputs {
					c[n] = X
				}

				c[in] = Zero

				out = append(out, c)
			}
		}

	case "NOT":

		if desired == One {
			out = append(
				out,
				Cube{
					g.Inputs[0]: Zero,
					g.Output:    One,
				},
			)
		}

		if desired == Zero {
			out = append(
				out,
				Cube{
					g.Inputs[0]: One,
					g.Output:    Zero,
				},
			)
		}
	}

	return out
}

// ============================================================
// ПЕЧАТЬ D-КУБОВ
// ============================================================

func printDTable(g Gate) {
	fmt.Printf(
		"\nD-кубы F%s (%s)\n",
		strings.TrimPrefix(g.Name, "F"),
		g.Type,
	)

	for _, in := range g.Inputs {
		fmt.Printf("%4d", poleNo[in])
	}

	fmt.Printf("%4d\n", poleNo[g.Output])

	fmt.Println(strings.Repeat("----", len(g.Inputs)+1))

	for i, c := range dCubes(g) {

		fmt.Printf("%d) ", i+1)

		for _, in := range g.Inputs {
			fmt.Printf("%4s", c[in])
		}

		fmt.Printf("%4s\n", c[g.Output])
	}
}

// ============================================================
// ПОИСК ВСЕХ D-ПУТЕЙ
// ============================================================

func findPaths(start string) [][]Gate {
	var result [][]Gate

	var dfs func(
		string,
		[]Gate,
		map[string]bool,
	)

	dfs = func(
		signal string,
		path []Gate,
		seen map[string]bool,
	) {
		if signal == "F6" {
			result = append(
				result,
				append([]Gate{}, path...),
			)
			return
		}

		for i := range gates {

			g := gates[i]

			found := false

			for _, in := range g.Inputs {
				if in == signal {
					found = true
					break
				}
			}

			if !found || seen[g.Output] {
				continue
			}

			nextSeen := map[string]bool{}

			for k, v := range seen {
				nextSeen[k] = v
			}

			nextSeen[g.Output] = true

			dfs(
				g.Output,
				append(path, g),
				nextSeen,
			)
		}
	}

	dfs(
		start,
		nil,
		map[string]bool{
			start: true,
		},
	)

	return result
}

// ============================================================
// ОДИН ШАГ ПРЯМОГО D-ПРОХОЖДЕНИЯ
// ============================================================

func propagateOne(
	c Cube,
	g Gate,
) ([]Cube, []int, []Cube) {

	var signal string
	var polarity Value

	for _, in := range g.Inputs {

		v := c[in]

		if v == D || v == DB {
			signal = in
			polarity = v
			break
		}
	}

	if signal == "" {
		return nil, nil, nil
	}

	var result []Cube
	var ids []int
	var used []Cube

	for i, dc := range dCubes(g) {

		if dc[signal] != polarity {
			continue
		}

		n, ok := merge(c, dc)

		if !ok {
			continue
		}

		if n[g.Output] != D &&
			n[g.Output] != DB {
			continue
		}

		result = append(result, n)
		ids = append(ids, i+1)
		used = append(used, copyCube(dc))
	}

	return result, ids, used
}

// ============================================================
// ПРЯМОЙ D-ПРОХОД
// fault → ... → F6
// ============================================================

func forward(
	path []Gate,
	start Cube,
) (Cube, []TraceStep, bool) {

	var rec func(
		int,
		Cube,
		[]TraceStep,
	) (Cube, []TraceStep, bool)

	rec = func(
		index int,
		current Cube,
		log []TraceStep,
	) (Cube, []TraceStep, bool) {

		if index == len(path) {
			return current, log, true
		}

		g := path[index]

		candidates, ids, usedCubes :=
			propagateOne(current, g)

		for k, next := range candidates {

			nextLog := append(
				[]TraceStep{},
				log...,
			)

			nextLog = append(
				nextLog,
				TraceStep{
					Before: copyCube(current),
					Desc: fmt.Sprintf(
						"%s: куб + D-куб №%d",
						g.Name,
						ids[k],
					),
					UsedCube: copyCube(usedCubes[k]),
					Cube:     copyCube(next),
				},
			)

			if result, resultLog, ok :=
				rec(index+1, next, nextLog); ok {

				return result, resultLog, true
			}
		}

		return nil, nil, false
	}

	return rec(0, start, nil)
}

// ============================================================
// ПОИСК ЗАВИСИМОСТЕЙ ОБРАТНОГО ПРОХОДА
//
// Например:
// F4 → F5 → F6
//
// F6 имеет дополнительный вход F1
// F5 имеет дополнительный вход F2
// F4 имеет дополнительный вход F3
//
// Поэтому:
// F1, F2, F3
//
// Для F2:
//
// F2 → F5 → F6
//
// F6 → F1
// F5 → F4
// F4 → F3
//
// Поэтому с учётом зависимости F3 → F4:
//
// F1, F3, F4
// ============================================================

func collectReverseGates(path []Gate) []Gate {

	// Все элементы прямого D-пути.
	onPath := map[string]bool{}

	for _, g := range path {
		onPath[g.Output] = true
	}

	// Сначала собираем непосредственные боковые входы.
	var roots []string

	for i := len(path) - 1; i >= 0; i-- {

		g := path[i]

		for _, in := range g.Inputs {

			// Если вход является выходом другого элемента
			// и этот элемент не находится на D-пути,
			// это боковая ветвь, которую надо раскрыть.
			ig := gateOf(in)

			if ig == nil {
				continue
			}

			if onPath[ig.Output] {
				continue
			}

			if !containsString(roots, ig.Output) {
				roots = append(roots, ig.Output)
			}
		}
	}

	// Рекурсивно добавляем зависимости боковых элементов.
	needed := map[string]bool{}

	var collect func(string)

	collect = func(name string) {

		if needed[name] {
			return
		}

		needed[name] = true

		g := gateOf(name)

		if g == nil {
			return
		}

		for _, in := range g.Inputs {

			dep := gateOf(in)

			if dep == nil {
				continue
			}

			if onPath[dep.Output] {
				continue
			}

			collect(dep.Output)
		}
	}

	for _, root := range roots {
		collect(root)
	}

	// Теперь строим топологический порядок:
	// сначала зависимости, потом элемент, который ими пользуется.
	var result []Gate
	visited := map[string]bool{}

	var visit func(string)

	visit = func(name string) {

		if visited[name] {
			return
		}

		g := gateOf(name)

		if g == nil {
			return
		}

		visited[name] = true

		for _, in := range g.Inputs {

			dep := gateOf(in)

			if dep == nil {
				continue
			}

			if onPath[dep.Output] {
				continue
			}

			if needed[dep.Output] {
				visit(dep.Output)
			}
		}

		result = append(result, *g)
	}

	for _, root := range roots {
		visit(root)
	}

	return result
}

func containsString(
	values []string,
	value string,
) bool {

	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}

// ============================================================
// ОБРАТНОЕ РАСКРЫТИЕ ОДНОГО ЭЛЕМЕНТА
// ============================================================

func expandGate(
	c Cube,
	g Gate,
	desired Value,
) ([]Cube, []int, []Cube) {

	var options []Cube

	if desired == D || desired == DB {

		for _, dc := range dCubes(g) {

			if dc[g.Output] == desired {
				options = append(options, dc)
			}
		}

	} else {

		options = singularCubes(
			g,
			desired,
		)
	}

	var result []Cube
	var ids []int
	var used []Cube

	for i, option := range options {

		n, ok := merge(c, option)

		if !ok {
			continue
		}

		// Выход элемента больше не нужен:
		// мы раскрываем его через входы.
		delete(n, g.Output)

		result = append(result, n)
		ids = append(ids, i+1)
		used = append(used, copyCube(option))
	}

	return result, ids, used
}

// ============================================================
// ОБРАТНЫЙ ПРОХОД
//
// Здесь НЕ идём F6,F5,F4,F3,F2,F1.
//
// Сначала берём конкретный D-путь,
// затем определяем его боковые ветви.
//
// Например:
//
// fault F4:
//
// прямой:
// F4 → F5 → F6
//
// обратные ветви:
// F1, F2, F3
//
// fault F2:
//
// прямой:
// F2 → F5 → F6
//
// обратные ветви:
// F1, F3, F4
// ============================================================

func backwardSearch(
	start Cube,
	f Fault,
	path []Gate,
) (Cube, []TraceStep, bool) {

	reverseGates := collectReverseGates(path)

	if len(reverseGates) == 0 {
		return start, nil, true
	}

	var rec func(
		int,
		Cube,
		[]TraceStep,
	) (Cube, []TraceStep, bool)

	rec = func(
		index int,
		current Cube,
		log []TraceStep,
	) (Cube, []TraceStep, bool) {

		if index == len(reverseGates) {
			return current, log, true
		}

		g := reverseGates[index]

		desired, ok := current[g.Output]

		if !ok {
			// Если значение пока не определено,
			// раскрывать нечего.
			return rec(
				index+1,
				current,
				log,
			)
		}

		options, ids, usedCubes :=
			expandGate(
				current,
				g,
				desired,
			)

		for k, next := range options {

			nextLog := append(
				[]TraceStep{},
				log...,
			)

			nextLog = append(
				nextLog,
				TraceStep{
					Before: copyCube(current),

					Desc: fmt.Sprintf(
						"%s=%s: обратный ход + сингулярный/D-куб №%d",
						g.Name,
						desired,
						ids[k],
					),

					UsedCube: copyCube(usedCubes[k]),
					Cube:     copyCube(next),
				},
			)

			if result, resultLog, ok :=
				rec(index+1, next, nextLog); ok {

				return result, resultLog, true
			}
		}

		return nil, nil, false
	}

	return rec(
		0,
		start,
		nil,
	)
}

// ============================================================
// ВЫВОД ШАГА
// ============================================================

func printStep(
	number int,
	s TraceStep,
) {

	fmt.Printf(
		"%2d. ",
		number,
	)

	printCubeInline(s.Before)

	fmt.Println()

	fmt.Printf(
		"    %s\n",
		s.Desc,
	)

	fmt.Print(
		"    ВЗЯТЫЙ_КУБ: ",
	)

	printCubeInline(s.UsedCube)

	fmt.Println()

	fmt.Print(
		"    -> ",
	)

	printCube(s.Cube)

	fmt.Print(
		"       ",
	)

	printCubeInline(s.Cube)

	fmt.Println()
}

// ============================================================
// ВХОДНАЯ ЧАСТЬ ИТОГОВОГО КУБА
// ============================================================

func inputSequence(c Cube) string {

	var b strings.Builder

	for _, p := range poles[:7] {

		v := c[p]

		if v == "" {
			v = X
		}

		b.WriteString(string(v))
	}

	return b.String()
}

func finalSequence(c Cube) string {

	var b strings.Builder

	for _, p := range poles {

		v := c[p]

		if v == "" {
			v = X
		}

		b.WriteString(string(v))
	}

	return b.String()
}

// ============================================================
// ВЫБОР НЕИСПРАВНОСТИ
// ============================================================

func chooseFault() Fault {

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ВЫБОР НЕИСПРАВНОСТИ")
	fmt.Println("============================================================")

	number := 1

	for _, p := range poles {

		fmt.Printf(
			"%2d. %s = 0\n",
			number,
			p,
		)

		number++

		fmt.Printf(
			"%2d. %s = 1\n",
			number,
			p,
		)

		number++
	}

	fmt.Print("\nВведите номер: ")

	reader := bufio.NewReader(os.Stdin)

	text, _ := reader.ReadString('\n')

	value, err :=
		strconv.Atoi(
			strings.TrimSpace(text),
		)

	if err != nil ||
		value < 1 ||
		value > 26 {

		panic("Неверный номер")
	}

	return Fault{
		Node: poles[(value-1)/2],

		StuckAt: (value - 1) % 2,
	}
}

// ============================================================
// ПЕЧАТЬ ПРЯМОГО ПУТИ
// ============================================================

func printForwardPath(
	f Fault,
	path []Gate,
) {

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ПРЯМОЙ D-ПРОХОД")
	fmt.Println("============================================================")

	fmt.Printf(
		"%s",
		f.Node,
	)

	for _, g := range path {
		fmt.Printf(
			" -> %s",
			g.Output,
		)
	}

	fmt.Println()
}

// ============================================================
// ПЕЧАТЬ ОБРАТНОГО ПОРЯДКА
// ============================================================

func printReversePath(
	path []Gate,
) {

	reverseGates :=
		collectReverseGates(path)

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ОБРАТНЫЙ ПРОХОД")
	fmt.Println("============================================================")

	if len(reverseGates) == 0 {
		fmt.Println("Нет боковых ветвей.")
		return
	}

	for i, g := range reverseGates {

		if i > 0 {
			fmt.Print(" -> ")
		}

		fmt.Print(g.Name)
	}

	fmt.Println()
}

// ============================================================
// MAIN
// ============================================================

func main() {

	fmt.Println("============================================================")
	fmt.Println("ЛР2 — D-АЛГОРИТМ — ВАРИАНТ 9")
	fmt.Println("============================================================")

	fmt.Println(
		"Схема: F1=AND, F2=NOT, F3=OR, F4=AND, F5=NAND, F6=AND",
	)

	fmt.Println(
		"Поля: 1..7 — входы x1..x7; 8..13 — F1..F6",
	)

	// --------------------------------------------------------
	// D-КУБЫ
	// --------------------------------------------------------

	fmt.Println()
	fmt.Println("ТАБЛИЦЫ D-КУБОВ")

	for _, g := range gates {
		printDTable(g)
	}

	// --------------------------------------------------------
	// НЕИСПРАВНОСТЬ
	// --------------------------------------------------------

	fault := chooseFault()

	fmt.Printf(
		"\nНЕИСПРАВНОСТЬ: %s stuck-at-%d\n",
		fault.Node,
		fault.StuckAt,
	)

	// --------------------------------------------------------
	// ПРИМИТИВНЫЙ D-КУБ
	// --------------------------------------------------------

	current := primitive(fault)

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ПОШАГОВОЕ ПОСТРОЕНИЕ")
	fmt.Println("============================================================")

	fmt.Printf(
		"%2d. Примитивный D-куб (%s stuck-at-%d)\n    ",
		1,
		fault.Node,
		fault.StuckAt,
	)

	printCube(current)

	// --------------------------------------------------------
	// ПРЯМОЙ D-ПУТЬ
	// --------------------------------------------------------

	var selectedPath []Gate
	var forwardLog []TraceStep

	success := false

	// F6 уже является выходом.
	if fault.Node == "F6" {

		selectedPath = nil
		success = true

	} else {

		allPaths := findPaths(fault.Node)

		for _, path := range allPaths {

			result, log, ok :=
				forward(
					path,
					current,
				)

			if !ok {
				continue
			}

			current = result
			selectedPath = path
			forwardLog = log
			success = true

			break
		}
	}

	if !success {

		fmt.Println()
		fmt.Println("Не удалось построить D-путь.")
		return
	}

	// --------------------------------------------------------
	// ПЕЧАТАЕМ ПРЯМОЙ ПУТЬ
	// --------------------------------------------------------

	printForwardPath(
		fault,
		selectedPath,
	)

	step := 2

	for _, s := range forwardLog {

		printStep(
			step,
			s,
		)

		step++
	}

	// --------------------------------------------------------
	// D-ЧАСТЬ ПОСЛЕ ПРЯМОГО ПРОХОДА
	// --------------------------------------------------------

	dPart := copyCube(current)

	// --------------------------------------------------------
	// ОБРАТНЫЙ ПОРЯДОК
	// --------------------------------------------------------

	printReversePath(
		selectedPath,
	)

	// --------------------------------------------------------
	// ОБРАТНОЕ РАСКРЫТИЕ
	// --------------------------------------------------------

	backResult,
		backLog,
		ok :=
		backwardSearch(
			current,
			fault,
			selectedPath,
		)

	if !ok {

		fmt.Println()
		fmt.Println(
			"Не удалось выполнить обратную фазу.",
		)

		return
	}

	for _, s := range backLog {

		printStep(
			step,
			s,
		)

		step++
	}

	// --------------------------------------------------------
	// СБОРКА ИТОГОВОГО КУБА
	// --------------------------------------------------------

	result := copyCube(dPart)

	for _, s := range backLog {

		var mergeOK bool

		result, mergeOK =
			merge(
				result,
				s.Cube,
			)

		if !mergeOK {

			fmt.Println()
			fmt.Println(
				"Не удалось собрать итоговый куб.",
			)

			return
		}
	}

	// backResult содержит результат обратной фазы.
	// Используем его как дополнительную проверку.
	_ = backResult

	// --------------------------------------------------------
	// ИТОГ
	// --------------------------------------------------------

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ИТОГОВЫЙ КУБ")
	fmt.Println("============================================================")

	printCube(result)

	fmt.Println()
	fmt.Println(
		"Полный куб:",
		finalSequence(result),
	)

	fmt.Println()
	fmt.Println(
		"Первые 7 позиций (x1...x7):",
	)

	fmt.Println(
		inputSequence(result),
	)
}
