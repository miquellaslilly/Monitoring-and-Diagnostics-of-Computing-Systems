package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ============================================================
// ИНДЕКСЫ ПОЛЮСОВ
// ============================================================

const (
	Ix1 = iota
	Ix2
	Ix3
	Ix4
	Ix5
	Ix6
	Ix7
	IF1
	IF2
	IF3
	IF4
	IF5
	IF6
	PoleCount
)

var poles = []string{
	"x1", "x2", "x3", "x4", "x5", "x6", "x7",
	"F1", "F2", "F3", "F4", "F5", "F6",
}

// ============================================================
// ЗНАЧЕНИЯ
// ============================================================

type Val string

const (
	V0 Val = "0"
	V1 Val = "1"
	VX Val = "X"
)

type Cube struct {
	Good   [PoleCount]Val
	Faulty [PoleCount]Val
}

func newCube() Cube {
	c := Cube{}
	for i := 0; i < PoleCount; i++ {
		c.Good[i] = VX
		c.Faulty[i] = VX
	}
	return c
}

func (c Cube) copy() Cube {
	return Cube{Good: c.Good, Faulty: c.Faulty}
}

// ============================================================
// СХЕМА
// ============================================================

type Gate struct {
	Name, Type string
	Inputs     []int
	Output     int
}

type Fault struct {
	Node    int
	StuckAt int
}

type TraceStep struct {
	Before   Cube
	Desc     string
	UsedCube Cube
	Cube     Cube
}

var gates = []Gate{
	{"F1", "AND", []int{Ix1, Ix2}, IF1},
	{"F2", "NOT", []int{Ix3}, IF2},
	{"F3", "OR", []int{Ix5, Ix6}, IF3},
	{"F4", "AND", []int{Ix4, IF3, Ix7}, IF4},
	{"F5", "NAND", []int{IF2, IF4}, IF5},
	{"F6", "AND", []int{IF1, IF5}, IF6},
}

func gateOf(out int) *Gate {
	for i := range gates {
		if gates[i].Output == out {
			return &gates[i]
		}
	}
	return nil
}

// ============================================================
// РАБОТА С КУБОМ
// ============================================================

func setPair(c *Cube, idx int, good, faulty Val) bool {
	curG := c.Good[idx]
	curF := c.Faulty[idx]

	if curG != VX && good != VX && curG != good {
		return false
	}
	if curF != VX && faulty != VX && curF != faulty {
		return false
	}

	if good != VX {
		c.Good[idx] = good
	}
	if faulty != VX {
		c.Faulty[idx] = faulty
	}
	return true
}

func mergeCube(a, b Cube) (Cube, bool) {
	r := a.copy()
	for i := 0; i < PoleCount; i++ {
		if !setPair(&r, i, b.Good[i], b.Faulty[i]) {
			return Cube{}, false
		}
	}
	return r, true
}

// ============================================================
// ПЕЧАТЬ
// ============================================================

func pairToSymbol(good, faulty Val) string {
	if good == V1 && faulty == V0 {
		return "d"
	}
	if good == V0 && faulty == V1 {
		return "d'"
	}
	if good == V1 && faulty == V1 {
		return "1"
	}
	if good == V0 && faulty == V0 {
		return "0"
	}
	return "X"
}

func symbolAt(c Cube, idx int) string {
	return pairToSymbol(c.Good[idx], c.Faulty[idx])
}

func printCubeInline(c Cube) {
	for i := 0; i < PoleCount; i++ {
		fmt.Print(symbolAt(c, i))
	}
}

func printCube(c Cube) {
	for i := 0; i < PoleCount; i++ {
		fmt.Printf("%2s ", symbolAt(c, i))
	}
	fmt.Println()
}

// ============================================================
// ПРИМИТИВНЫЙ КУБ НЕИСПРАВНОСТИ
// ============================================================

func primitive(f Fault) Cube {
	g := gateOf(f.Node)

	var goodOut Val
	if f.StuckAt == 0 {
		goodOut = V1
	} else {
		goodOut = V0
	}

	if g == nil {
		p := newCube()
		if f.StuckAt == 0 {
			setPair(&p, f.Node, V1, V0)
		} else {
			setPair(&p, f.Node, V0, V1)
		}
		return p
	}

	opts := singularCubes(*g, goodOut)
	if len(opts) == 0 {
		p := newCube()
		if f.StuckAt == 0 {
			setPair(&p, f.Node, V1, V0)
		} else {
			setPair(&p, f.Node, V0, V1)
		}
		return p
	}

	res := opts[0].copy()
	res.Good[g.Output] = VX
	res.Faulty[g.Output] = VX
	if f.StuckAt == 0 {
		setPair(&res, g.Output, V1, V0)
	} else {
		setPair(&res, g.Output, V0, V1)
	}
	return res
}

// ============================================================
// D-КУБЫ
// ============================================================

type sym struct {
	Idx   int
	Value string
}

func cubeFromSymbols(items ...sym) Cube {
	c := newCube()
	for _, it := range items {
		switch it.Value {
		case "0":
			setPair(&c, it.Idx, V0, V0)
		case "1":
			setPair(&c, it.Idx, V1, V1)
		case "D":
			setPair(&c, it.Idx, V1, V0)
		case "DB":
			setPair(&c, it.Idx, V0, V1)
		case "X":
			// X — не задаём
		}
	}
	return c
}

func dCubes(g Gate) []Cube {
	out := []Cube{}

	switch g.Type {

	case "AND":
		for _, dv := range []string{"D", "DB"} {
			for i := range g.Inputs {
				items := []sym{}
				for j, n := range g.Inputs {
					if i == j {
						items = append(items, sym{n, dv})
					} else {
						items = append(items, sym{n, "1"})
					}
				}
				items = append(items, sym{g.Output, dv})
				out = append(out, cubeFromSymbols(items...))
			}
		}

	case "OR":
		for _, dv := range []string{"D", "DB"} {
			for i := range g.Inputs {
				items := []sym{}
				for j, n := range g.Inputs {
					if i == j {
						items = append(items, sym{n, dv})
					} else {
						items = append(items, sym{n, "0"})
					}
				}
				items = append(items, sym{g.Output, dv})
				out = append(out, cubeFromSymbols(items...))
			}
		}

	case "NAND":
		for _, dv := range []string{"D", "DB"} {
			outSym := "D"
			if dv == "D" {
				outSym = "DB"
			}
			for i := range g.Inputs {
				items := []sym{}
				for j, n := range g.Inputs {
					if i == j {
						items = append(items, sym{n, dv})
					} else {
						items = append(items, sym{n, "1"})
					}
				}
				items = append(items, sym{g.Output, outSym})
				out = append(out, cubeFromSymbols(items...))
			}
		}

	case "NOT":
		out = append(out,
			cubeFromSymbols(
				sym{g.Inputs[0], "D"},
				sym{g.Output, "DB"},
			),
			cubeFromSymbols(
				sym{g.Inputs[0], "DB"},
				sym{g.Output, "D"},
			),
		)
	}

	return out
}

// ============================================================
// СИНГУЛЯРНЫЕ КУБЫ
// ============================================================

func singularCubes(g Gate, desired Val) []Cube {
	out := []Cube{}
	desiredSym := string(desired)

	switch g.Type {

	case "AND":
		if desired == V1 {
			items := []sym{{g.Output, desiredSym}}
			for _, in := range g.Inputs {
				items = append(items, sym{in, "1"})
			}
			out = append(out, cubeFromSymbols(items...))
		}
		if desired == V0 {
			for _, in := range g.Inputs {
				items := []sym{{g.Output, desiredSym}}
				for _, n := range g.Inputs {
					if n == in {
						items = append(items, sym{n, "0"})
					} else {
						items = append(items, sym{n, "X"})
					}
				}
				out = append(out, cubeFromSymbols(items...))
			}
		}

	case "OR":
		if desired == V0 {
			items := []sym{{g.Output, desiredSym}}
			for _, in := range g.Inputs {
				items = append(items, sym{in, "0"})
			}
			out = append(out, cubeFromSymbols(items...))
		}
		if desired == V1 {
			for _, in := range g.Inputs {
				items := []sym{{g.Output, desiredSym}}
				for _, n := range g.Inputs {
					if n == in {
						items = append(items, sym{n, "1"})
					} else {
						items = append(items, sym{n, "X"})
					}
				}
				out = append(out, cubeFromSymbols(items...))
			}
		}

	case "NAND":
		if desired == V0 {
			items := []sym{{g.Output, desiredSym}}
			for _, in := range g.Inputs {
				items = append(items, sym{in, "1"})
			}
			out = append(out, cubeFromSymbols(items...))
		}
		if desired == V1 {
			for _, in := range g.Inputs {
				items := []sym{{g.Output, desiredSym}}
				for _, n := range g.Inputs {
					if n == in {
						items = append(items, sym{n, "0"})
					} else {
						items = append(items, sym{n, "X"})
					}
				}
				out = append(out, cubeFromSymbols(items...))
			}
		}

	case "NOT":
		if desired == V1 {
			out = append(out,
				cubeFromSymbols(
					sym{g.Inputs[0], "0"},
					sym{g.Output, "1"},
				),
			)
		}
		if desired == V0 {
			out = append(out,
				cubeFromSymbols(
					sym{g.Inputs[0], "1"},
					sym{g.Output, "0"},
				),
			)
		}
	}

	return out
}

// ============================================================
// ПЕЧАТЬ ТАБЛИЦ D-КУБОВ
// ============================================================

func printDTable(g Gate) {
	fmt.Printf("\nD-кубы %s (%s)\n", g.Name, g.Type)

	for _, in := range g.Inputs {
		fmt.Printf("%6d", in+1)
	}
	fmt.Printf("%6d\n", g.Output+1)

	fmt.Println(strings.Repeat("------", len(g.Inputs)+1))

	for i, c := range dCubes(g) {
		fmt.Printf(" %d) ", i+1)
		for _, in := range g.Inputs {
			fmt.Printf("%6s", c.Good[in])
		}
		fmt.Printf("%6s\n", c.Good[g.Output])

		fmt.Print("    ")
		for _, in := range g.Inputs {
			fmt.Printf("%6s", c.Faulty[in])
		}
		fmt.Printf("%6s\n", c.Faulty[g.Output])
	}
}

// ============================================================
// ПОИСК ПУТЕЙ
// ============================================================

func findPaths(start int) [][]Gate {
	var result [][]Gate
	var dfs func(int, []Gate, [PoleCount]bool)

	dfs = func(signal int, path []Gate, seen [PoleCount]bool) {
		if signal == IF6 {
			result = append(result, append([]Gate{}, path...))
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
			nextSeen := seen
			nextSeen[g.Output] = true
			dfs(g.Output, append(path, g), nextSeen)
		}
	}

	var seen [PoleCount]bool
	seen[start] = true
	dfs(start, nil, seen)
	return result
}

// ============================================================
// ОДИН ШАГ ПРЯМОГО ПРОХОДА
// ============================================================

func hasDiff(c Cube, idx int) bool {
	g := c.Good[idx]
	f := c.Faulty[idx]
	if g == VX || f == VX {
		return false
	}
	return g != f
}

func diffDir(c Cube, idx int) string {
	g := c.Good[idx]
	f := c.Faulty[idx]
	if g == V1 && f == V0 {
		return "D"
	}
	if g == V0 && f == V1 {
		return "DB"
	}
	return ""
}

func propagateOne(c Cube, g Gate) ([]Cube, []int, []Cube) {
	var signal int = -1
	var dir string

	for _, in := range g.Inputs {
		if hasDiff(c, in) {
			signal = in
			dir = diffDir(c, in)
			break
		}
	}
	if signal == -1 {
		return nil, nil, nil
	}

	var result []Cube
	var ids []int
	var used []Cube

	for i, dc := range dCubes(g) {
		if diffDir(dc, signal) != dir {
			continue
		}
		n, ok := mergeCube(c, dc)
		if !ok {
			continue
		}
		if !hasDiff(n, g.Output) {
			continue
		}
		result = append(result, n)
		ids = append(ids, i+1)
		used = append(used, dc.copy())
	}
	return result, ids, used
}

// ============================================================
// ПРЯМОЙ ПРОХОД
// ============================================================

func forward(path []Gate, start Cube) (Cube, []TraceStep, bool) {
	var rec func(int, Cube, []TraceStep) (Cube, []TraceStep, bool)

	rec = func(index int, current Cube, log []TraceStep) (Cube, []TraceStep, bool) {
		if index == len(path) {
			return current, log, true
		}
		g := path[index]
		candidates, ids, usedCubes := propagateOne(current, g)
		for k, next := range candidates {
			nextLog := append([]TraceStep{}, log...)
			nextLog = append(nextLog, TraceStep{
				Before:   current.copy(),
				Desc:     fmt.Sprintf("%s: куб + D-куб №%d", g.Name, ids[k]),
				UsedCube: usedCubes[k].copy(),
				Cube:     next.copy(),
			})
			if result, resultLog, ok := rec(index+1, next, nextLog); ok {
				return result, resultLog, true
			}
		}
		return Cube{}, nil, false
	}

	return rec(0, start, nil)
}

// ============================================================
// СБОР БОКОВЫХ ВЕТВЕЙ
// ============================================================

// collectReverseGates — элементы боковых ветвей.
// Неисправный элемент исключается (fault.Node).
func collectReverseGates(path []Gate, faultNode int) []Gate {
	var onPath [PoleCount]bool
	for _, g := range path {
		onPath[g.Output] = true
	}
	// Неисправный элемент тоже «на пути» — не раскрываем его.
	onPath[faultNode] = true

	var roots []int
	for i := len(path) - 1; i >= 0; i-- {
		g := path[i]
		for _, in := range g.Inputs {
			ig := gateOf(in)
			if ig == nil || onPath[ig.Output] {
				continue
			}
			if !containsInt(roots, ig.Output) {
				roots = append(roots, ig.Output)
			}
		}
	}

	var needed [PoleCount]bool
	var collect func(int)
	collect = func(name int) {
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
			if dep == nil || onPath[dep.Output] {
				continue
			}
			collect(dep.Output)
		}
	}
	for _, root := range roots {
		collect(root)
	}

	var result []Gate
	var visited [PoleCount]bool
	var visit func(int)
	visit = func(name int) {
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
			if dep == nil || onPath[dep.Output] {
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

func containsInt(values []int, value int) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// ============================================================
// РАСКРЫТИЕ ОДНОГО ЭЛЕМЕНТА
// ============================================================

func allInputsKnown(c Cube, g Gate) bool {
	for _, in := range g.Inputs {
		if c.Good[in] == VX || c.Faulty[in] == VX {
			return false
		}
	}
	return true
}

func expandGate(c Cube, g Gate) ([]Cube, []int, []Cube) {
	outG := c.Good[g.Output]
	outF := c.Faulty[g.Output]

	if outG == VX && outF == VX {
		return nil, nil, nil
	}

	if allInputsKnown(c, g) {
		res := c.copy()
		res.Good[g.Output] = VX
		res.Faulty[g.Output] = VX
		return []Cube{res}, []int{0}, []Cube{res}
	}

	var options []Cube

	if outG != VX && outF != VX && outG != outF {
		dir := "D"
		if outG == V0 && outF == V1 {
			dir = "DB"
		}
		for _, dc := range dCubes(g) {
			if diffDir(dc, g.Output) == dir {
				options = append(options, dc)
			}
		}
	} else {
		desired := outG
		if desired == VX {
			desired = outF
		}
		options = singularCubes(g, desired)
	}

	var result []Cube
	var ids []int
	var used []Cube

	for i, option := range options {
		n, ok := mergeCube(c, option)
		if !ok {
			continue
		}
		n.Good[g.Output] = VX
		n.Faulty[g.Output] = VX

		result = append(result, n)
		ids = append(ids, i+1)
		used = append(used, option.copy())
	}
	return result, ids, used
}

// ============================================================
// ОБРАТНЫЙ ПРОХОД
// ============================================================

func backwardSearch(start Cube, f Fault, path []Gate) (Cube, []TraceStep, bool) {
	reverseGates := collectReverseGates(path, f.Node)
	if len(reverseGates) == 0 {
		return start, nil, true
	}

	var rec func(int, Cube, []TraceStep) (Cube, []TraceStep, bool)
	rec = func(index int, current Cube, log []TraceStep) (Cube, []TraceStep, bool) {
		if index == len(reverseGates) {
			return current, log, true
		}
		g := reverseGates[index]

		gOut := current.Good[g.Output]
		fOut := current.Faulty[g.Output]

		if gOut == VX && fOut == VX {
			return rec(index+1, current, log)
		}

		options, ids, usedCubes := expandGate(current, g)
		for k, next := range options {
			nextLog := append([]TraceStep{}, log...)

			desc := fmt.Sprintf("%s: обратный ход + куб №%d",
				g.Name, ids[k])
			if ids[k] == 0 {
				desc = fmt.Sprintf("%s: обратный ход (все входы уже заданы)",
					g.Name)
			}

			nextLog = append(nextLog, TraceStep{
				Before:   current.copy(),
				Desc:     desc,
				UsedCube: usedCubes[k].copy(),
				Cube:     next.copy(),
			})
			if result, resultLog, ok := rec(index+1, next, nextLog); ok {
				return result, resultLog, true
			}
		}
		return Cube{}, nil, false
	}

	return rec(0, start, nil)
}

// ============================================================
// ПЕЧАТЬ ШАГА
// ============================================================

func printStep(number int, s TraceStep) {
	fmt.Printf("%2d. ", number)
	printCubeInline(s.Before)
	fmt.Println()

	fmt.Printf("    %s\n", s.Desc)

	fmt.Print("    ВЗЯТЫЙ_КУБ: ")
	printCubeInline(s.UsedCube)
	fmt.Println()

	fmt.Print("    -> ")
	printCube(s.Cube)
	fmt.Print("       ")
	printCubeInline(s.Cube)
	fmt.Println()
}

// ============================================================
// ИТОГОВЫЕ СТРОКИ
// ============================================================

func inputSequence(c Cube) string {
	var b strings.Builder
	for i := 0; i < 7; i++ {
		g := c.Good[i]
		f := c.Faulty[i]

		if g == VX && f == VX {
			b.WriteString("X")
		} else if g == f {
			b.WriteString(string(g))
		} else if g != VX && f != VX {
			b.WriteString(string(g))
		} else {
			b.WriteString("X")
		}
	}
	return b.String()
}

func finalSequence(c Cube) string {
	var b strings.Builder
	for i := 0; i < PoleCount; i++ {
		b.WriteString(symbolAt(c, i))
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
		fmt.Printf("%2d. %s = 0\n", number, p)
		number++
		fmt.Printf("%2d. %s = 1\n", number, p)
		number++
	}

	fmt.Print("\nВведите номер: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || value < 1 || value > 26 {
		panic("Неверный номер")
	}
	return Fault{
		Node:    (value - 1) / 2,
		StuckAt: (value - 1) % 2,
	}
}

// ============================================================
// ПЕЧАТЬ ПУТЕЙ
// ============================================================

func printForwardPath(f Fault, path []Gate) {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ПРЯМОЙ D-ПРОХОД")
	fmt.Println("============================================================")
	fmt.Printf("%s", poles[f.Node])
	for _, g := range path {
		fmt.Printf(" -> %s", g.Name)
	}
	fmt.Println()
}

func printReversePath(path []Gate, faultNode int) {
	reverseGates := collectReverseGates(path, faultNode)
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
	fmt.Println("Схема: F1=AND, F2=NOT, F3=OR, F4=AND, F5=NAND, F6=AND")
	fmt.Println("Поля: 1..7 — входы x1..x7; 8..13 — F1..F6")

	fmt.Println()
	fmt.Println("ТАБЛИЦЫ D-КУБОВ (первая строка куба — good, вторая — faulty)")

	for _, g := range gates {
		printDTable(g)
	}

	fault := chooseFault()
	fmt.Printf("\nНЕИСПРАВНОСТЬ: %s stuck-at-%d\n",
		poles[fault.Node], fault.StuckAt)

	current := primitive(fault)

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ПОШАГОВОЕ ПОСТРОЕНИЕ")
	fmt.Println("============================================================")

	fmt.Printf("%2d. Примитивный D-куб (%s stuck-at-%d)\n    ",
		1, poles[fault.Node], fault.StuckAt)
	printCube(current)

	var selectedPath []Gate
	var forwardLog []TraceStep
	success := false

	if fault.Node == IF6 {
		selectedPath = nil
		success = true
	} else {
		allPaths := findPaths(fault.Node)
		for _, path := range allPaths {
			result, log, ok := forward(path, current)
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

	printForwardPath(fault, selectedPath)

	step := 2
	for _, s := range forwardLog {
		printStep(step, s)
		step++
	}

	dPart := current.copy()

	printReversePath(selectedPath, fault.Node)

	backResult, backLog, ok := backwardSearch(current, fault, selectedPath)
	if !ok {
		fmt.Println()
		fmt.Println("Не удалось выполнить обратную фазу.")
		return
	}
	_ = backResult

	for _, s := range backLog {
		printStep(step, s)
		step++
	}

	// Сборка итогового куба.
	result := dPart.copy()
	for _, s := range backLog {
		var mergeOK bool
		result, mergeOK = mergeCube(result, s.Cube)
		if !mergeOK {
			fmt.Println()
			fmt.Println("!!! Не удалось собрать итоговый куб.")
			fmt.Println("    result:")
			printCubeInline(result)
			fmt.Println()
			fmt.Println("    s.Cube:")
			printCubeInline(s.Cube)
			fmt.Println()
			return
		}
	}

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("ИТОГОВЫЙ КУБ")
	fmt.Println("============================================================")
	printCube(result)

	fmt.Println()
	fmt.Println("Полный куб:", finalSequence(result))
	fmt.Println()
	fmt.Println("Первые 7 позиций (x1...x7):")
	fmt.Println(inputSequence(result))
}
