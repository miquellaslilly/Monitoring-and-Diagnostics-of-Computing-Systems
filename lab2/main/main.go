package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Value int

const (
	Zero Value = iota
	One
	X
	D
	DB // D-bar
)

func (v Value) String() string {
	switch v {
	case Zero:
		return "0"
	case One:
		return "1"
	case X:
		return "X"
	case D:
		return "D"
	case DB:
		return "D'"
	default:
		return "?"
	}
}

type Cube map[string]Value

func copyCube(c Cube) Cube {
	result := make(Cube)

	for k, v := range c {
		result[k] = v
	}

	return result
}

func valueOf(c Cube, name string) Value {
	if v, ok := c[name]; ok {
		return v
	}

	return X
}

func setValue(c Cube, name string, value Value) {
	c[name] = value
}

func printCube(c Cube, title string) {
	nodes := []string{
		"x1", "x2", "x3", "x4", "x5", "x6", "x7",
		"F1", "F2", "F3", "F4", "F5", "F6",
	}

	fmt.Println(title)

	for _, node := range nodes {
		fmt.Printf("%-3s ", node)
	}

	fmt.Println()

	for _, node := range nodes {
		fmt.Printf("%-3s ", valueOf(c, node))
	}

	fmt.Println()
	fmt.Println()
}

// ------------------------------------------------------------
// D-пересечение
// ------------------------------------------------------------

func intersect(a, b Cube) (Cube, bool) {
	result := copyCube(a)

	for node, bValue := range b {
		aValue := valueOf(result, node)

		if aValue == X {
			result[node] = bValue
			continue
		}

		if bValue == X {
			continue
		}

		if aValue != bValue {
			return nil, false
		}
	}

	return result, true
}

// ------------------------------------------------------------
// Логические элементы
// ------------------------------------------------------------

type Gate struct {
	Name   string
	Type   string
	Inputs []string
}

// Схема варианта 9:
//
// F1 = x1 AND x2
// F2 = NOT x3
// F3 = x5 OR x6
// F4 = x4 AND F3 AND x7
// F5 = NAND(F2, F4)
// F6 = F1 AND F5
//

var gates = []Gate{
	{
		Name:   "F1",
		Type:   "AND",
		Inputs: []string{"x1", "x2"},
	},
	{
		Name:   "F2",
		Type:   "NOT",
		Inputs: []string{"x3"},
	},
	{
		Name:   "F3",
		Type:   "OR",
		Inputs: []string{"x5", "x6"},
	},
	{
		Name:   "F4",
		Type:   "AND",
		Inputs: []string{"x4", "F3", "x7"},
	},
	{
		Name:   "F5",
		Type:   "NAND",
		Inputs: []string{"F2", "F4"},
	},
	{
		Name:   "F6",
		Type:   "AND",
		Inputs: []string{"F1", "F5"},
	},
}

var gateByName = map[string]Gate{
	"F1": gates[0],
	"F2": gates[1],
	"F3": gates[2],
	"F4": gates[3],
	"F5": gates[4],
	"F6": gates[5],
}

// Следующий элемент на пути распространения D.

var nextGate = map[string]string{
	"F1": "F6",
	"F2": "F5",
	"F3": "F4",
	"F4": "F5",
	"F5": "F6",
	"F6": "",
}

// ------------------------------------------------------------
// Сингулярные кубы
//
// Возвращаются условия на входах элемента,
// обеспечивающие требуемое исправное значение выхода.
// ------------------------------------------------------------

func singularCubes(g Gate, output Value) []Cube {
	var result []Cube

	switch g.Type {

	case "AND":
		if output == One {
			// AND = 1 -> все входы 1
			c := Cube{}

			for _, input := range g.Inputs {
				c[input] = One
			}

			result = append(result, c)
		}

		if output == Zero {
			// AND = 0 -> хотя бы один вход 0
			for _, input := range g.Inputs {
				c := Cube{}

				for _, other := range g.Inputs {
					c[other] = X
				}

				c[input] = Zero
				result = append(result, c)
			}
		}

	case "OR":
		if output == Zero {
			// OR = 0 -> все входы 0
			c := Cube{}

			for _, input := range g.Inputs {
				c[input] = Zero
			}

			result = append(result, c)
		}

		if output == One {
			// OR = 1 -> хотя бы один вход 1
			for _, input := range g.Inputs {
				c := Cube{}

				for _, other := range g.Inputs {
					c[other] = X
				}

				c[input] = One
				result = append(result, c)
			}
		}

	case "NAND":
		// NAND = NOT(AND)

		if output == Zero {
			// NAND = 0 -> все входы 1
			c := Cube{}

			for _, input := range g.Inputs {
				c[input] = One
			}

			result = append(result, c)
		}

		if output == One {
			// NAND = 1 -> хотя бы один вход 0
			for _, input := range g.Inputs {
				c := Cube{}

				for _, other := range g.Inputs {
					c[other] = X
				}

				c[input] = Zero
				result = append(result, c)
			}
		}

	case "NOT":
		if output == One {
			result = append(result, Cube{
				g.Inputs[0]: Zero,
			})
		}

		if output == Zero {
			result = append(result, Cube{
				g.Inputs[0]: One,
			})
		}
	}

	return result
}

// ------------------------------------------------------------
// D-куб распространения
//
// Например:
//
// AND:
//
// D 1 -> D
// 1 D -> D
//
// OR:
//
// D 0 -> D
// 0 D -> D
//
// NAND:
//
// D 1 -> D'
// 1 D -> D'
//
// NOT:
//
// D -> D'
// D' -> D
// ------------------------------------------------------------

func propagationCube(g Gate, dValue Value, inputIndex int) Cube {
	c := Cube{}

	// Значение на входе, по которому распространяется D.
	c[g.Inputs[inputIndex]] = dValue

	switch g.Type {

	case "AND":
		for i, input := range g.Inputs {
			if i != inputIndex {
				c[input] = One
			}
		}

		c[g.Name] = dValue

	case "OR":
		for i, input := range g.Inputs {
			if i != inputIndex {
				c[input] = Zero
			}
		}

		c[g.Name] = dValue

	case "NAND":
		for i, input := range g.Inputs {
			if i != inputIndex {
				c[input] = One
			}
		}

		if dValue == D {
			c[g.Name] = DB
		} else {
			c[g.Name] = D
		}

	case "NOT":
		if dValue == D {
			c[g.Name] = DB
		} else {
			c[g.Name] = D
		}
	}

	return c
}

// ------------------------------------------------------------
// Примитивный D-куб
//
// SA0:
//
// исправное значение = 1
// неисправное       = 0
//
// => D
//
// SA1:
//
// исправное значение = 0
// неисправное       = 1
//
// => D'
// ------------------------------------------------------------

func primitiveCube(faultGate string, stuckAt int) Cube {
	c := Cube{}

	if stuckAt == 0 {
		c[faultGate] = D
	} else {
		c[faultGate] = DB
	}

	return c
}

// ------------------------------------------------------------
// Построение пути от неисправности к выходу
// ------------------------------------------------------------

func buildPath(start string) []string {
	var path []string

	current := start

	for current != "" {
		path = append(path, current)
		current = nextGate[current]
	}

	return path
}

// ------------------------------------------------------------
// Поиск позиции входа элемента
// ------------------------------------------------------------

func inputIndex(g Gate, input string) int {
	for i, name := range g.Inputs {
		if name == input {
			return i
		}
	}

	return -1
}

// ------------------------------------------------------------
// D-проход
// ------------------------------------------------------------

func dPass(faultGate string, cube Cube) (Cube, []string, bool) {
	path := buildPath(faultGate)

	var log []string

	currentCube := copyCube(cube)

	fmt.Println("Путь D-прохода:")

	for i := 0; i < len(path); i++ {
		fmt.Print(path[i])

		if i != len(path)-1 {
			fmt.Print(" -> ")
		}
	}

	fmt.Println()
	fmt.Println()

	for i := 0; i < len(path)-1; i++ {
		currentGateName := path[i]
		nextGateName := path[i+1]

		currentD := valueOf(currentCube, currentGateName)

		nextGateStruct := gateByName[nextGateName]

		position := inputIndex(nextGateStruct, currentGateName)

		if position == -1 {
			return nil, log, false
		}

		localCube := propagationCube(
			nextGateStruct,
			currentD,
			position,
		)

		fmt.Printf(
			"Переход %s -> %s\n",
			currentGateName,
			nextGateName,
		)

		printCube(
			localCube,
			"Локальный D-куб:",
		)

		newCube, ok := intersect(currentCube, localCube)

		if !ok {
			fmt.Println("D-пересечение: Ø")
			fmt.Println("Путь не может быть активизирован.")
			return nil, log, false
		}

		fmt.Println("D-пересечение выполнено успешно.")
		printCube(newCube, "Полученный куб:")

		currentCube = newCube

		log = append(
			log,
			fmt.Sprintf(
				"%s -> %s",
				currentGateName,
				nextGateName,
			),
		)
	}

	return currentCube, log, true
}

// ------------------------------------------------------------
// Проверка, находится ли элемент на D-пути.
// ------------------------------------------------------------

func isOnPath(gateName string, path []string) bool {
	for _, p := range path {
		if p == gateName {
			return true
		}
	}

	return false
}

// ------------------------------------------------------------
// Обратная фаза.
//
// Идём:
//
// F6 -> F5 -> F4 -> F3 -> F2 -> F1
//
// Для обычного значения выхода выбираем сингулярные кубы.
// Для неисправного элемента, содержащего D/D', выбираем
// сингулярные кубы для исправного значения выхода.
// ------------------------------------------------------------

func reversePhase(
	cube Cube,
	faultGate string,
	path []string,
) []Cube {

	reverseGates := []string{
		"F6",
		"F5",
		"F4",
		"F3",
		"F2",
		"F1",
	}

	var result []Cube

	var process func(int, Cube)

	process = func(index int, current Cube) {

		if index == len(reverseGates) {
			result = append(result, copyCube(current))
			return
		}

		name := reverseGates[index]
		gate := gateByName[name]

		output := valueOf(current, name)

		// Если выход X, ограничивать его не требуется.
		if output == X {
			process(index+1, current)
			return
		}

		// D/D' на элементах пути уже получили условия
		// во время D-прохода.
		//
		// Исключение — сам неисправный элемент:
		// для него нужно выполнить активизацию неисправности.
		if (output == D || output == DB) &&
			name != faultGate {

			process(index+1, current)
			return
		}

		var requiredOutput Value

		switch output {
		case D:
			// D = исправное 1 / неисправное 0
			requiredOutput = One

		case DB:
			// D' = исправное 0 / неисправное 1
			requiredOutput = Zero

		default:
			requiredOutput = output
		}

		candidates := singularCubes(gate, requiredOutput)

		if len(candidates) == 0 {
			return
		}

		for number, candidate := range candidates {

			fmt.Printf(
				"Обратная фаза: %s, вариант %d\n",
				name,
				number+1,
			)

			printCube(
				candidate,
				"Сингулярный куб:",
			)

			merged, ok := intersect(current, candidate)

			if !ok {
				fmt.Println("Пересечение: Ø")
				fmt.Println("Вариант отбрасывается.")
				fmt.Println()

				continue
			}

			fmt.Println("Пересечение успешно.")
			printCube(
				merged,
				"Результат:",
			)

			process(index+1, merged)
		}
	}

	process(0, cube)

	return result
}

// ------------------------------------------------------------
// Получение тестовых наборов из куба
// ------------------------------------------------------------

var primaryInputs = []string{
	"x1", "x2", "x3", "x4",
	"x5", "x6", "x7",
}

func cubeToTests(c Cube) []string {

	var tests []string

	var generate func(int, []int)

	generate = func(index int, bits []int) {

		if index == len(primaryInputs) {
			var builder strings.Builder

			for _, bit := range bits {
				builder.WriteString(strconv.Itoa(bit))
			}

			tests = append(tests, builder.String())
			return
		}

		value := valueOf(c, primaryInputs[index])

		switch value {

		case Zero:
			generate(index+1, append(bits, 0))

		case One:
			generate(index+1, append(bits, 1))

		case X:
			generate(index+1, append(bits, 0))
			generate(index+1, append(bits, 1))

		default:
			// На первичных входах D/D' возникать не должны.
			return
		}
	}

	generate(0, nil)

	return tests
}

// ------------------------------------------------------------
// Моделирование исправной схемы
// ------------------------------------------------------------

func evaluate(inputs []int) int {

	x1 := inputs[0]
	x2 := inputs[1]
	x3 := inputs[2]
	x4 := inputs[3]
	x5 := inputs[4]
	x6 := inputs[5]
	x7 := inputs[6]

	f1 := x1 & x2
	f2 := 1 - x3
	f3 := x5 | x6
	f4 := x4 & f3 & x7
	f5 := 1 - (f2 & f4)
	f6 := f1 & f5

	return f6
}

// ------------------------------------------------------------
// Моделирование схемы с неисправностью
// ------------------------------------------------------------

func evaluateFault(
	inputs []int,
	faultGate string,
	stuckAt int,
) int {

	x1 := inputs[0]
	x2 := inputs[1]
	x3 := inputs[2]
	x4 := inputs[3]
	x5 := inputs[4]
	x6 := inputs[5]
	x7 := inputs[6]

	f1 := x1 & x2
	f2 := 1 - x3
	f3 := x5 | x6
	f4 := x4 & f3 & x7
	f5 := 1 - (f2 & f4)
	f6 := f1 & f5

	switch faultGate {

	case "F1":
		f1 = stuckAt

	case "F2":
		f2 = stuckAt

	case "F3":
		f3 = stuckAt

	case "F4":
		f4 = stuckAt

	case "F5":
		f5 = stuckAt

	case "F6":
		f6 = stuckAt
	}

	// Повторное распространение изменения
	// через последующие элементы.

	switch faultGate {

	case "F1":
		f6 = f1 & f5

	case "F2":
		f5 = 1 - (f2 & f4)
		f6 = f1 & f5

	case "F3":
		f4 = x4 & f3 & x7
		f5 = 1 - (f2 & f4)
		f6 = f1 & f5

	case "F4":
		f5 = 1 - (f2 & f4)
		f6 = f1 & f5

	case "F5":
		f6 = f1 & f5

	case "F6":
		// F6 уже неисправен.
	}

	return f6
}

// ------------------------------------------------------------
// Проверка полученного теста
// ------------------------------------------------------------

func verifyTest(test string, faultGate string, stuckAt int) bool {

	inputs := make([]int, 7)

	for i, char := range test {
		inputs[i] = int(char - '0')
	}

	good := evaluate(inputs)
	faulty := evaluateFault(
		inputs,
		faultGate,
		stuckAt,
	)

	return good != faulty
}

// ------------------------------------------------------------
// Основная функция
// ------------------------------------------------------------

func main() {

	fmt.Println("====================================================")
	fmt.Println("ЛАБОРАТОРНАЯ РАБОТА №2")
	fmt.Println("Метод активизации многомерного пути")
	fmt.Println("Вариант 9")
	fmt.Println("====================================================")
	fmt.Println()

	fmt.Println("Схема:")
	fmt.Println("F1 = И")
	fmt.Println("F2 = НЕ")
	fmt.Println("F3 = ИЛИ")
	fmt.Println("F4 = И")
	fmt.Println("F5 = И-НЕ")
	fmt.Println("F6 = И")
	fmt.Println()

	var faultGate string
	var stuckAt int

	fmt.Print("Введите неисправный элемент (F1...F6): ")
	fmt.Scan(&faultGate)

	fmt.Print("Введите константную неисправность (0 или 1): ")
	fmt.Scan(&stuckAt)

	if _, ok := gateByName[faultGate]; !ok {
		fmt.Println("Ошибка: неизвестный элемент.")
		os.Exit(1)
	}

	if stuckAt != 0 && stuckAt != 1 {
		fmt.Println("Ошибка: неисправность должна быть 0 или 1.")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("====================================================")
	fmt.Printf(
		"НЕИСПРАВНОСТЬ: %s ≡ %d\n",
		faultGate,
		stuckAt,
	)
	fmt.Println("====================================================")
	fmt.Println()

	// --------------------------------------------------------
	// 1. Примитивный D-куб
	// --------------------------------------------------------

	fmt.Println("[1] ПРИМИТИВНЫЙ D-КУБ")
	fmt.Println()

	cube := primitiveCube(
		faultGate,
		stuckAt,
	)

	printCube(
		cube,
		"Примитивный D-куб:",
	)

	// --------------------------------------------------------
	// 2. D-проход
	// --------------------------------------------------------

	fmt.Println("[2] D-ПРОХОД")
	fmt.Println()

	path := buildPath(faultGate)

	fmt.Print("Путь: ")

	for i, node := range path {
		if i > 0 {
			fmt.Print(" -> ")
		}

		fmt.Print(node)
	}

	fmt.Println()
	fmt.Println()

	dCube, _, ok := dPass(
		faultGate,
		cube,
	)

	if !ok {
		fmt.Println("Не удалось построить D-проход.")
		return
	}

	fmt.Println("D достиг выходного элемента F6.")
	fmt.Println()

	// --------------------------------------------------------
	// 3. Обратная фаза
	// --------------------------------------------------------

	fmt.Println("[3] ОБРАТНАЯ ФАЗА")
	fmt.Println()

	finalCubes := reversePhase(
		dCube,
		faultGate,
		path,
	)

	if len(finalCubes) == 0 {
		fmt.Println("Решение не найдено.")
		return
	}

	// --------------------------------------------------------
	// 4. Итоговые кубы
	// --------------------------------------------------------

	fmt.Println("====================================================")
	fmt.Println("[4] ИТОГОВЫЕ КУБЫ")
	fmt.Println("====================================================")
	fmt.Println()

	for i, c := range finalCubes {

		fmt.Printf("Куб №%d\n", i+1)

		printCube(
			c,
			"",
		)
	}

	// --------------------------------------------------------
	// 5. Получение тестовых наборов
	// --------------------------------------------------------

	fmt.Println("====================================================")
	fmt.Println("[5] КОНКРЕТНЫЕ ТЕСТОВЫЕ НАБОРЫ")
	fmt.Println("====================================================")
	fmt.Println()

	testNumber := 1

	for i, c := range finalCubes {

		fmt.Printf("Для куба №%d:\n", i+1)

		tests := cubeToTests(c)

		for _, test := range tests {

			valid := verifyTest(
				test,
				faultGate,
				stuckAt,
			)

			status := "OK"

			if !valid {
				status = "ОШИБКА"
			}

			fmt.Printf(
				"T%-3d = %s   [%s]\n",
				testNumber,
				test,
				status,
			)

			testNumber++
		}

		fmt.Println()
	}

	fmt.Println("====================================================")
	fmt.Println("Работа алгоритма завершена.")
	fmt.Println("====================================================")
}
